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

type CloudPrincipalResolver struct {
	vedro.CloudPrincipal

	KubeClient client.Client
	Logger     logr.Logger

	Condition metav1.Condition
	Error     error
}

func (o *CloudPrincipalResolver) IsOk() bool {
	return o.Error == nil
}

func (o *CloudPrincipalResolver) IsReady() (*metav1.Condition, bool) {
	return isReady(o.Generation, o.Status.Conditions)
}

func (o *CloudPrincipalResolver) Resolve(
	ctx context.Context,
	name types.NamespacedName,
) {
	o.Error = nil
	o.CloudPrincipal = vedro.CloudPrincipal{}
	o.Logger.V(1).Info("getting CloudPrincipal")

	o.Condition = metav1.Condition{
		Type:   conditions.TypeReady,
		Status: metav1.ConditionFalse,
	}

	err := o.KubeClient.Get(ctx, name, &o.CloudPrincipal)
	if err != nil {
		o.Error = err

		if apierrors.IsNotFound(err) {
			o.Logger.Info("CloudPrincipal not found")
			o.Condition.Reason = conditions.ReasonCloudPrincipalNotFound
			o.Condition.Message = "CloudPrincipal was not found"
			return
		}
		o.Logger.Error(err, "failed to get CloudPrincipal")

		o.Condition.Reason = conditions.ReasonCloudPrincipalGetFailed
		o.Condition.Message = err.Error()
	}
}
