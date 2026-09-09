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

type CloudPrincipalAuthResolver struct {
	vedro.CloudPrincipalAuth

	KubeClient client.Client
	Logger     logr.Logger

	Condition metav1.Condition
	Error     error
}

func (o *CloudPrincipalAuthResolver) IsOk() bool {
	return o.Error == nil
}

func (o *CloudPrincipalAuthResolver) IsBeingDeleted() bool {
	return !o.DeletionTimestamp.IsZero()
}

func (o *CloudPrincipalAuthResolver) IsProvisioned() bool {
	return o.Status.Applied != nil
}

func (o *CloudPrincipalAuthResolver) IsReady() (*metav1.Condition, bool) {
	return isReady(o.Generation, o.Status.Conditions)
}

func (o *CloudPrincipalAuthResolver) ShouldBeRetained() bool {
	return (o.Spec.StaticCredentials != nil && o.Spec.StaticCredentials.DeletionPolicy == vedro.DeletionPolicyRetain) ||
		(o.Spec.WorkloadIdentity != nil && o.Spec.WorkloadIdentity.DeletionPolicy == vedro.DeletionPolicyRetain)

}

func (o *CloudPrincipalAuthResolver) ShouldBeDeleted() bool {
	return (o.Spec.StaticCredentials != nil && o.Spec.StaticCredentials.DeletionPolicy == vedro.DeletionPolicyDelete) ||
		(o.Spec.WorkloadIdentity != nil && o.Spec.WorkloadIdentity.DeletionPolicy == vedro.DeletionPolicyDelete)
}

func (o *CloudPrincipalAuthResolver) IsReferenced(
	ctx context.Context,
) (bool, error) {
	return false, nil
}

func (o *CloudPrincipalAuthResolver) Resolve(
	ctx context.Context,
	name types.NamespacedName,
) {
	o.Error = nil
	o.CloudPrincipalAuth = vedro.CloudPrincipalAuth{}
	o.Logger.V(1).Info("getting CloudPrincipal")

	o.Condition = metav1.Condition{
		Type:   conditions.TypeReady,
		Status: metav1.ConditionFalse,
	}

	err := o.KubeClient.Get(ctx, name, &o.CloudPrincipalAuth)
	if err != nil {
		o.Error = err

		if apierrors.IsNotFound(err) {
			o.Logger.Info("CloudPrincipalAuth not found")
			o.Condition.Reason = conditions.ReasonCloudPrincipalAuthNotFound
			o.Condition.Message = "CloudPrincipalAuth was not found"
			return
		}
		o.Logger.Error(err, "failed to get CloudPrincipalAuth")

		o.Condition.Reason = conditions.ReasonCloudPrincipalAuthGetFailed
		o.Condition.Message = err.Error()
	}
}
