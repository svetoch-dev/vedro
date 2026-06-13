package resolvers

import (
	"context"

	"github.com/go-logr/logr"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/conditions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceResolver interface {
	Populate(ctx context.Context, name types.NamespacedName)
	IsOk() bool
}

// ProviderConfig
type ProviderConfigResolver struct {
	vedrov1alpha1.ProviderConfig

	KubeClient client.Client
	Log        logr.Logger

	Condition metav1.Condition
	Error     error
}

func (o *ProviderConfigResolver) IsOk() bool {
	return o.Error == nil
}

func (o *ProviderConfigResolver) Resolve(
	ctx context.Context,
	name types.NamespacedName,
) {
	o.Error = nil
	o.ProviderConfig = vedrov1alpha1.ProviderConfig{}
	o.Log.V(1).Info("getting ProviderConfig", "name", name.String())

	o.Condition = metav1.Condition{
		Type: conditions.TypeProviderConfigReady,
	}

	err := o.KubeClient.Get(ctx, name, &o.ProviderConfig)

	if err != nil {
		o.Error = err
		o.Condition.Status = metav1.ConditionFalse

		if apierrors.IsNotFound(err) {
			o.Log.Info("ProviderConfig not found", "name", name.String())
			o.Condition.Reason = conditions.ReasonProviderConfigNotFound
			o.Condition.Message = "ProviderConfig was not found"
			return
		}
		o.Log.Error(err, "failed to get ProviderConfig", "name", name.String())

		o.Condition.Reason = conditions.ReasonProviderConfigGetFailed
		o.Condition.Message = err.Error()
		return
	}
}
