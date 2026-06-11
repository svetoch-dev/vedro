package resolvers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BucketResolver struct {
	Object vedrov1alpha1.Bucket

	KubeClient client.Client
	Log        logr.Logger

	Condition metav1.Condition
	Error     error
}

func (b *BucketResolver) IsOk() bool {
	return b.Error == nil
}

func (b *BucketResolver) SetStatusCondition(condition metav1.Condition) {
	meta.SetStatusCondition(&b.Object.Status.Conditions, condition)
}

func (b *BucketResolver) Resolve(
	ctx context.Context,
	name types.NamespacedName,
) {
	b.Error = nil
	b.Object = vedrov1alpha1.Bucket{}
	b.Log.V(1).Info("getting bucket", "name", name.String())

	b.Condition = metav1.Condition{
		Type: "Ready",
	}

	err := b.KubeClient.Get(ctx, name, &b.Object)
	if err != nil {
		b.Error = err
		b.Condition.Status = metav1.ConditionFalse

		if apierrors.IsNotFound(err) {
			b.Log.Info("Bucket not found", "name", name.String())
			b.Condition.Reason = "BucketNotFound"
			b.Condition.Message = fmt.Sprintf("Bucket %q was not found", name.Name)
			return
		}
		b.Log.Error(err, "failed to get Bucket", "name", name.String())

		b.Condition.Reason = "BucketGetFailed"
		b.Condition.Message = err.Error()
		return
	}

	b.Condition.Status = metav1.ConditionTrue
	b.Condition.Reason = "BucketFound"
	b.Condition.Message = fmt.Sprintf("Bucket %q was found", name.Name)
}
