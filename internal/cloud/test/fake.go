package cloudtest

import (
	"context"
	"errors"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
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
		StorageClass:           storageClass,
		PublicAccessPrevention: helpers.Ptr(false),
		Lifecycle:              &vedro.BucketLifecycle{},
		Versioning:             &vedro.BucketVersioning{Enabled: false},
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
	Name       string
	Generation int64
}

type FakeBucketAPI struct {
	Attrs     *cloud.BucketAttrs
	AttrsErr  error
	CreateErr error
	UpdateErr error
	DeleteErr error
	Created   *cloud.BucketAttrs
	Updated   *cloud.BucketPatch

	Deleted         bool
	Query           *storage.Query
	ObjectIterator  *FakeObjectIterator
	ObjectDeleteErr error

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
	f.Query = &storage.Query{Versions: true}
	if f.ObjectIterator != nil {
		for {
			attrs, err := f.ObjectIterator.Next()
			if errors.Is(err, iterator.Done) {
				return nil
			}
			if err != nil {
				return err
			}
			if err := process(cloud.ObjectVersion{Name: attrs.Name, Version: attrs.Generation}); err != nil {
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

func (f *FakeBucketAPI) recordDeletedObject(name string, generation int64) {
	f.deletedObjectsMu.Lock()
	defer f.deletedObjectsMu.Unlock()
	f.deletedObjects = append(f.deletedObjects, DeletedObject{Name: name, Generation: generation})
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
) error {
	f.Created = &attrs
	if f.CreateErr != nil {
		return f.CreateErr
	}
	f.Attrs = &attrs
	return nil
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

	return f.Attrs, nil
}

// FakeObjectIterator is a test iterator over object attributes.
type FakeObjectIterator struct {
	Attrs []*storage.ObjectAttrs
	Err   error
	index int
}

func (f *FakeObjectIterator) Next() (*storage.ObjectAttrs, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.index >= len(f.Attrs) {
		return nil, iterator.Done
	}
	attrs := f.Attrs[f.index]
	f.index++
	return attrs, nil
}
