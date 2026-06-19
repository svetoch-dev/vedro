package gcp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/gammazero/workerpool"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/validation"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

var (
	dualRegionPattern   = regexp.MustCompile(`^[A-Z]+[0-9]+$`)
	storageClassMapping = map[vedrov1alpha1.BucketStorageClass]string{
		vedrov1alpha1.BucketStorageClassStandard:         "STANDARD",
		vedrov1alpha1.BucketStorageClassInfrequentAccess: "NEARLINE",
		vedrov1alpha1.BucketStorageClassArchive:          "ARCHIVE",
	}
	publicAccessPreventionMapping = map[bool]storage.PublicAccessPrevention{
		false: storage.PublicAccessPreventionInherited,
		true:  storage.PublicAccessPreventionEnforced,
	}
	lifeCycleMapping = map[vedrov1alpha1.BucketLifecycleAction]string{
		vedrov1alpha1.BucketLifecycleActionDelete: storage.DeleteAction,
	}
)

func validateGCSLocation(location string) *validation.ValidationResult {
	normalized := strings.ToUpper(location)

	// Known multi-regions.
	switch normalized {
	case "US", "EU", "ASIA":
		v := validation.Valid()
		return &v
	}

	// Allow predefined dual-region IDs like NAM4, EUR4, ASIA1.
	if dualRegionPattern.MatchString(normalized) {
		v := validation.Valid()
		return &v
	}

	return nil
}

func validateGCSName(name string) *validation.ValidationResult {
	if strings.HasPrefix(name, "goog") || strings.Contains(name, "google") {
		v := validation.Invalid("bucket name must not use reserved Google-related names")
		return &v
	}

	return nil
}

func setGCSLabels(
	desiredLabels map[string]string,
	actualLabels map[string]string,
	attrs *storage.BucketAttrsToUpdate,
) {
	if attrs == nil {
		return
	}

	for k, v := range desiredLabels {
		attrs.SetLabel(k, v)
	}

	for k := range actualLabels {
		if _, ok := desiredLabels[k]; !ok {
			attrs.DeleteLabel(k)
		}
	}
}

func convertToGCSLifeCycle(lifeCycle vedrov1alpha1.BucketLifecycleSpec) (storage.Lifecycle, error) {
	gcsLifeCycle := storage.Lifecycle{}

	for index, rule := range lifeCycle.Rules {
		if !rule.Enabled {
			continue
		}

		actionType, ok := lifeCycleMapping[rule.Action]
		if !ok {
			return gcsLifeCycle, fmt.Errorf(
				"lifecycle.rules[%d].action %s doesn't map to any GCS lifecycle action",
				index,
				rule.Action,
			)
		}

		gcsAction := storage.LifecycleAction{
			Type: actionType,
		}

		var condition *storage.LifecycleCondition

		if rule.AgeDays != nil {
			condition = &storage.LifecycleCondition{
				AgeInDays: *rule.AgeDays,
			}
		}

		if condition != nil {
			gcsLifeCycle.Rules = append(gcsLifeCycle.Rules, storage.LifecycleRule{
				Action:    gcsAction,
				Condition: *condition,
			})
		}
	}

	return gcsLifeCycle, nil
}

// Abstraction for tests
type storageClient interface {
	Bucket(name string) bucketHandle
}

type bucketHandle interface {
	Attrs(ctx context.Context) (*storage.BucketAttrs, error)
	Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error
	Update(ctx context.Context, uattrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error)
	Objects(ctx context.Context, q *storage.Query) objectIterator
	Delete(ctx context.Context) error
	Object(name string) objectHandle
}

type objectHandle interface {
	Generation(gen int64) objectHandle
	Delete(ctx context.Context) error
}

type objectIterator interface {
	Next() (*storage.ObjectAttrs, error)
}

type storageClientAdapter struct {
	client *storage.Client
}

func (a *storageClientAdapter) Bucket(name string) bucketHandle {
	return &realBucketHandle{bh: a.client.Bucket(name)}
}

type realBucketHandle struct {
	bh *storage.BucketHandle
}

func (r *realBucketHandle) Attrs(ctx context.Context) (*storage.BucketAttrs, error) {
	return r.bh.Attrs(ctx)
}

func (r *realBucketHandle) Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error {
	return r.bh.Create(ctx, projectID, attrs)
}

func (r *realBucketHandle) Update(ctx context.Context, uattrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error) {
	return r.bh.Update(ctx, uattrs)
}

func (r *realBucketHandle) Objects(ctx context.Context, q *storage.Query) objectIterator {
	return r.bh.Objects(ctx, q)
}

func (r *realBucketHandle) Delete(ctx context.Context) error {
	return r.bh.Delete(ctx)
}

func (r *realBucketHandle) Object(name string) objectHandle {
	return &realObjectHandle{oh: r.bh.Object(name)}
}

type realObjectHandle struct {
	oh *storage.ObjectHandle
}

func (r *realObjectHandle) Generation(gen int64) objectHandle {
	return &realObjectHandle{oh: r.oh.Generation(gen)}
}

func (r *realObjectHandle) Delete(ctx context.Context) error {
	return r.oh.Delete(ctx)
}

type Bucket struct {
	client    storageClient
	projectId string
}

func (b *Bucket) ValidateBucketSpec(bckt vedrov1alpha1.Bucket) validation.ValidationResult {
	spec := bckt.Spec

	v := validation.ValidateBucketNameImmutability(bckt)

	if !v.Valid {
		return v
	}

	v = validation.ValidateBucketLocation(spec.Location, validateGCSLocation)

	if !v.Valid {
		return v
	}

	bucketName := helpers.BucketNameFromCR(bckt)
	v = validation.ValidateBucketName(bucketName, validateGCSName)
	if !v.Valid {
		return v
	}

	return validation.Valid()
}

func (b *Bucket) EnsureBucket(ctx context.Context, bckt vedrov1alpha1.Bucket) (*cloud.BucketState, error) {
	spec := bckt.Spec
	status := bckt.Status

	bucketName := helpers.BucketNameFromCR(bckt)
	normalizedLocation := strings.ToUpper(spec.Location)

	bucket := b.client.Bucket(bucketName)

	storageClass, ok := storageClassMapping[spec.StorageClass]
	if !ok {
		return nil, fmt.Errorf("spec.StorageClass %s doesnt map to any bucket StorageClass", spec.StorageClass)
	}

	var publicAccessPrevention storage.PublicAccessPrevention
	if spec.PublicAccessPrevention != nil {
		publicAccessPrevention = publicAccessPreventionMapping[*spec.PublicAccessPrevention]
	}

	var versioning bool
	if spec.Versioning != nil {
		versioning = spec.Versioning.Enabled
	} else {
		versioning = false
	}

	var gcsLifeCycle storage.Lifecycle

	if spec.Lifecycle != nil {
		var err error
		gcsLifeCycle, err = convertToGCSLifeCycle(*spec.Lifecycle)
		if err != nil {
			return nil, err
		}
	}

	applied := vedrov1alpha1.BucketAppliedState{}
	if status.Applied != nil {
		applied = *status.Applied
	}

	attrs, err := bucket.Attrs(ctx)

	if errors.Is(err, storage.ErrBucketNotExist) {
		createAttrs := storage.BucketAttrs{
			Location:          spec.Location,
			StorageClass:      storageClass,
			Labels:            spec.Labels,
			VersioningEnabled: versioning,
		}

		applied.StorageClass = spec.StorageClass
		applied.Labels = spec.Labels
		applied.Versioning = &vedrov1alpha1.BucketVersioningSpec{Enabled: versioning}

		createAttrs.Lifecycle = gcsLifeCycle
		applied.Lifecycle = spec.Lifecycle

		if spec.PublicAccessPrevention != nil {
			createAttrs.PublicAccessPrevention = publicAccessPrevention
			applied.PublicAccessPrevention = spec.PublicAccessPrevention
		}

		if err := bucket.Create(ctx, b.projectId, &createAttrs); err != nil {
			return nil, fmt.Errorf("create bucket %q: %w", bucketName, err)
		}

		return &cloud.BucketState{
			ExternalName: bucketName,
			Location:     normalizedLocation,
			Applied:      &applied,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get bucket attrs %q: %w", bucketName, err)
	}

	if attrs.Location != normalizedLocation {
		return nil, fmt.Errorf(
			"bucket %q already exists in location %q, desired location is %q",
			bucketName,
			attrs.Location,
			normalizedLocation,
		)
	}

	updateAttrs := storage.BucketAttrsToUpdate{}
	needsUpdate := false

	if !maps.Equal(applied.Labels, spec.Labels) {
		setGCSLabels(spec.Labels, applied.Labels, &updateAttrs)
		applied.Labels = spec.Labels
		needsUpdate = true
	}

	if attrs.StorageClass != storageClass {
		updateAttrs.StorageClass = storageClass
		applied.StorageClass = spec.StorageClass
		needsUpdate = true
	}

	if attrs.VersioningEnabled != versioning {
		updateAttrs.VersioningEnabled = versioning
		applied.Versioning = &vedrov1alpha1.BucketVersioningSpec{Enabled: versioning}
		needsUpdate = true
	}

	if spec.PublicAccessPrevention != nil && attrs.PublicAccessPrevention != publicAccessPrevention {
		updateAttrs.PublicAccessPrevention = publicAccessPrevention
		applied.PublicAccessPrevention = spec.PublicAccessPrevention
		needsUpdate = true
	}

	if !reflect.DeepEqual(gcsLifeCycle, attrs.Lifecycle) {
		updateAttrs.Lifecycle = &gcsLifeCycle
		applied.Lifecycle = spec.Lifecycle
		needsUpdate = true
	}

	if needsUpdate {
		attrs, err = bucket.Update(ctx, updateAttrs)
		if err != nil {
			return nil, fmt.Errorf("update bucket %q: %w", bucketName, err)
		}
	}

	return &cloud.BucketState{
		ExternalName: bucketName,
		Location:     attrs.Location,
		Applied:      &applied,
	}, nil
}

func (b *Bucket) DeleteBucket(ctx context.Context, bckt vedrov1alpha1.Bucket) error {
	bucketName := helpers.BucketNameFromCR(bckt)

	// Extra check to be super sure
	if bckt.Spec.DeletionPolicy != vedrov1alpha1.DeletionPolicyDelete {
		return nil
	}

	bh := b.client.Bucket(bucketName)

	// err that we will return if object deletion fails
	var deleteObjectError error
	// error mutex for syncing concurrent changes to error var
	var errM sync.Mutex

	// New pool with workers equal number of CPU
	workers := runtime.NumCPU() - 1
	if workers < 1 {
		workers = 1
	}
	wp := workerpool.New(workers)

	// Semaphore channel allowing up to 2000 uncompleted deletion tasks in the queue
	sem := make(chan struct{}, 2000)

	it := bh.Objects(ctx, &storage.Query{
		Versions: true,
	})

	for {
		objAttrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}

		// queue task
		sem <- struct{}{}

		wp.Submit(func() {
			// dequeue task. Defer only accepts callable so wrap it in func
			defer func() { <-sem }()
			err := bh.Object(objAttrs.Name).Generation(objAttrs.Generation).Delete(ctx)
			if err != nil {
				errM.Lock()
				defer errM.Unlock()
				var gErr *googleapi.Error
				if errors.As(err, &gErr) && gErr.Code == 404 {
					return
				}
				if deleteObjectError == nil {
					deleteObjectError = err
				}

			}
		})
	}

	wp.StopWait()

	if deleteObjectError != nil {
		return fmt.Errorf("could not delete bucket because object deletion failed: %w", deleteObjectError)
	}

	err := bh.Delete(ctx)
	if err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) && gErr.Code == 404 {
			return nil
		}

		return fmt.Errorf("could not delete bucket because of error: %w", err)
	}

	return nil
}
