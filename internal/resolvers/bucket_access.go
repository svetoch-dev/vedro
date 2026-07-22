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

type BucketAccessResolver struct {
	vedro.BucketAccess

	KubeClient client.Client
	Logger     logr.Logger

	Condition metav1.Condition
	Error     error
}

func (o *BucketAccessResolver) IsOk() bool {
	return o.Error == nil
}

func (o *BucketAccessResolver) Resolve(
	ctx context.Context,
	name types.NamespacedName,
) {
	o.Error = nil
	o.BucketAccess = vedro.BucketAccess{}
	o.Logger.V(1).Info("getting BucketAccess")

	o.Condition = metav1.Condition{
		Type:   conditions.TypeReady,
		Status: metav1.ConditionFalse,
	}

	err := o.KubeClient.Get(ctx, name, &o.BucketAccess)
	if err != nil {
		o.Error = err

		if apierrors.IsNotFound(err) {
			o.Logger.Info("BucketAccess not found")
			o.Condition.Reason = conditions.ReasonCloudPrincipalNotFound
			o.Condition.Message = "BucketAccess was not found"
			return
		}
		o.Logger.Error(err, "failed to get BucketAccess")

		o.Condition.Reason = conditions.ReasonBucketAccessGetFailed
		o.Condition.Message = err.Error()
	}
}
