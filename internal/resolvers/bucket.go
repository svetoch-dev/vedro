package resolvers

import (
	"context"

	"github.com/go-logr/logr"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/conditions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BucketResolver struct {
	vedro.Bucket

	KubeClient client.Client
	Logger     logr.Logger

	Condition metav1.Condition
	Error     error
}

func (o *BucketResolver) IsOk() bool {
	return o.Error == nil
}

func (o *BucketResolver) IsBeingDeleted() bool {
	return !o.DeletionTimestamp.IsZero()
}

func (o *BucketResolver) IsReady() (*metav1.Condition, bool) {
	return isReady(o.Generation, o.Status.Conditions)
}

func (o *BucketResolver) ShouldBeRetained() bool {
	return o.Spec.DeletionPolicy == vedro.DeletionPolicyRetain
}

func (o *BucketResolver) ShouldBeDeleted() bool {
	return o.Spec.DeletionPolicy == vedro.DeletionPolicyDelete
}

func (o *BucketResolver) IsReferenced(
	ctx context.Context,
) (bool, error) {
	var bucketAccessList vedro.BucketAccessList
	if err := o.KubeClient.List(ctx, &bucketAccessList); err != nil {
		return false, err
	}

	for _, bucketAccess := range bucketAccessList.Items {
		if bucketAccess.Spec.BucketRef.Name == o.Name &&
			bucketAccess.Spec.BucketRef.Namespace == o.Namespace {
			return true, nil
		}
	}

	return false, nil
}

func (o *BucketResolver) Resolve(
	ctx context.Context,
	name types.NamespacedName,
) {
	o.Error = nil
	o.Bucket = vedro.Bucket{}
	o.Logger.V(1).Info("getting bucket")

	o.Condition = metav1.Condition{
		Type:   conditions.TypeReady,
		Status: metav1.ConditionFalse,
	}

	err := o.KubeClient.Get(ctx, name, &o.Bucket)
	if err != nil {
		o.Error = err

		if apierrors.IsNotFound(err) {
			o.Logger.Info("Bucket not found")
			o.Condition.Reason = conditions.ReasonBucketNotFound
			o.Condition.Message = "Bucket was not found"
			return
		}
		o.Logger.Error(err, "failed to get Bucket")

		o.Condition.Reason = conditions.ReasonBucketGetFailed
		o.Condition.Message = err.Error()
		return
	}
}
