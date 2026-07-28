package cloudtest

import (
	"context"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

func NewBucketCR(
	name string,
	location string,
	mods ...func(*vedro.Bucket),
) vedro.Bucket {
	b := vedro.Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: vedro.BucketSpec{
			ProviderRef: vedro.ProviderConfigReference{Name: "some-provider"},
			Location:    location,
		},
	}
	for _, m := range mods {
		m(&b)
	}
	return b
}

func NewBucketAttrs(
	name string,
	location string,
	storageClass vedro.BucketStorageClass,

	mods ...func(*vedro.BucketProperties),
) *cloud.BucketAttrs {
	properties := &vedro.BucketProperties{
		StorageClass: storageClass,
	}
	for _, mod := range mods {
		mod(properties)
	}
	return &cloud.BucketAttrs{
		Name:       name,
		Location:   location,
		Properties: properties,
	}
}

var Lifecycle = vedro.BucketLifecycle{
	Rules: []vedro.BucketLifecycleRule{
		{
			AgeDays: helpers.Ptr(int64(2)),
			Action:  vedro.BucketLifecycleActionDelete,
			Enabled: true,
		},
	},
}

type DeletedObject struct {
	Name    string
	Version *string
}

type FakeBucketAPI struct {
	Attrs              *cloud.BucketAttrs
	AttrsErr           error
	CreateErr          error
	UpdateErr          error
	DeleteErr          error
	CloseErr           error
	Created            *cloud.BucketAttrs
	Updated            *cloud.BucketPatch
	HasAccessInputs    []cloud.BucketAccessAttrs
	GrantAccessInputs  []cloud.BucketAccessAttrs
	RevokeAccessInputs []cloud.BucketAccessAttrs

	Deleted              bool
	CloseCalled          bool
	ProcessObjectsCalled bool
	ObjectIterator       *FakeObjectIterator
	ObjectDeleteErr      error

	deletedObjectsMu sync.Mutex
	deletedObjects   []DeletedObject
}

var _ cloud.BucketAPI = (*FakeBucketAPI)(nil)

func (f *FakeBucketAPI) DeleteBucket(ctx context.Context, _ string) error {
	f.Deleted = true
	return f.DeleteErr
}

func (f *FakeBucketAPI) ProcessObjects(
	ctx context.Context,
	_ string,
	process func(cloud.ObjectVersion) error,
) error {
	f.ProcessObjectsCalled = true
	if f.ObjectIterator != nil {
		for {
			object, ok, err := f.ObjectIterator.Next()
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			if err := process(object); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *FakeBucketAPI) DeleteObject(
	ctx context.Context,
	_ string,
	object cloud.ObjectVersion,
) error {
	f.recordDeletedObject(object.Name, object.Version)
	return f.ObjectDeleteErr
}

func (f *FakeBucketAPI) recordDeletedObject(name string, version *string) {
	f.deletedObjectsMu.Lock()
	defer f.deletedObjectsMu.Unlock()
	f.deletedObjects = append(f.deletedObjects, DeletedObject{Name: name, Version: version})
}

// GetDeletedObjects returns a copy of the objects deleted so far.
func (f *FakeBucketAPI) GetDeletedObjects() []DeletedObject {
	f.deletedObjectsMu.Lock()
	defer f.deletedObjectsMu.Unlock()
	out := make([]DeletedObject, len(f.deletedObjects))
	copy(out, f.deletedObjects)
	return out
}

func (f *FakeBucketAPI) GetBucket(ctx context.Context, _ string) (*cloud.BucketAttrs, error) {
	if f.AttrsErr != nil {
		return nil, f.AttrsErr
	}
	return f.Attrs, nil
}

func (f *FakeBucketAPI) CreateBucket(
	ctx context.Context,
	_ string,
	attrs cloud.BucketAttrs,
) (*cloud.BucketAttrs, error) {
	f.Created = &attrs
	if f.CreateErr != nil {
		return nil, f.CreateErr
	}
	f.Attrs = &attrs
	return &attrs, nil
}

func (f *FakeBucketAPI) HasAccess(
	ctx context.Context,
	access cloud.BucketAccessAttrs,
) (bool, error) {
	f.HasAccessInputs = append(f.HasAccessInputs, access)
	return false, nil
}

func (f *FakeBucketAPI) GrantAccess(
	ctx context.Context,
	access cloud.BucketAccessAttrs,
) error {
	f.GrantAccessInputs = append(f.GrantAccessInputs, access)
	return nil
}

func (f *FakeBucketAPI) RevokeAccess(
	ctx context.Context,
	access cloud.BucketAccessAttrs,
) error {
	f.RevokeAccessInputs = append(f.RevokeAccessInputs, access)
	return nil
}

func (f *FakeBucketAPI) Close(ctx context.Context) error {
	f.CloseCalled = true
	return f.CloseErr
}

func (f *FakeBucketAPI) UpdateBucket(
	ctx context.Context,
	_ string,
	patch cloud.BucketPatch,
) (*cloud.BucketAttrs, error) {
	f.Updated = &patch
	if f.UpdateErr != nil {
		return nil, f.UpdateErr
	}

	if f.Attrs.Properties == nil {
		f.Attrs.Properties = &vedro.BucketProperties{}
	}
	if patch.StorageClass.Set {
		f.Attrs.Properties.StorageClass = patch.StorageClass.Value
	}
	if patch.Labels.Set {
		f.Attrs.Properties.Labels = patch.Labels.Value
	}
	if patch.Versioning.Set {
		f.Attrs.Properties.Versioning = patch.Versioning.Value
	}
	if patch.PublicAccessPrevention.Set {
		f.Attrs.Properties.PublicAccessPrevention = patch.PublicAccessPrevention.Value
	}
	if patch.Lifecycle.Set {
		f.Attrs.Properties.Lifecycle = patch.Lifecycle.Value
	}
	if patch.CloudSpecificConfig.Set {
		f.Attrs.Properties.CloudSpecificConfig = patch.CloudSpecificConfig.Value
	}

	return f.Attrs, nil
}

// FakeObjectIterator is a test iterator over provider-neutral object versions.
type FakeObjectIterator struct {
	Objects []cloud.ObjectVersion
	Err     error
	index   int
}

func (f *FakeObjectIterator) Next() (cloud.ObjectVersion, bool, error) {
	if f.Err != nil {
		return cloud.ObjectVersion{}, false, f.Err
	}
	if f.index >= len(f.Objects) {
		return cloud.ObjectVersion{}, false, nil
	}
	object := f.Objects[f.index]
	f.index++
	return object, true, nil
}
