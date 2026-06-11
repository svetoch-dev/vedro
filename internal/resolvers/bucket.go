package resolvers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/conditions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BucketResolver struct {
	vedrov1alpha1.Bucket

	KubeClient client.Client
	Log        logr.Logger

	Condition metav1.Condition
	Error     error
}

func (o *BucketResolver) IsOk() bool {
	return o.Error == nil
}

func (o *BucketResolver) Resolve(
	ctx context.Context,
	name types.NamespacedName,
) {
	o.Error = nil
	o.Bucket = vedrov1alpha1.Bucket{}
	o.Log.V(1).Info("getting bucket", "name", name.String())

	o.Condition = metav1.Condition{
		Type: conditions.TypeReady,
	}

	err := o.KubeClient.Get(ctx, name, &o.Bucket)
	if err != nil {
		o.Error = err
		o.Condition.Status = metav1.ConditionFalse

		if apierrors.IsNotFound(err) {
			o.Log.Info("Bucket not found", "name", name.String())
			o.Condition.Reason = conditions.ReasonBucketNotFound
			o.Condition.Message = fmt.Sprintf("Bucket %q was not found", name.Name)
			return
		}
		o.Log.Error(err, "failed to get Bucket", "name", name.String())

		o.Condition.Reason = conditions.ReasonBucketGetFailed
		o.Condition.Message = err.Error()
		return
	}

	o.Condition.Status = metav1.ConditionTrue
	o.Condition.Reason = conditions.ReasonBucketFound
	o.Condition.Message = fmt.Sprintf("Bucket %q was found", name.Name)
}
