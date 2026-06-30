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

type ResourceResolver interface {
	Resolve(ctx context.Context, name types.NamespacedName)
	IsOk() bool
}

// ProviderConfig
type ProviderConfigResolver struct {
	vedro.ProviderConfig

	KubeClient client.Client
	Logger     logr.Logger

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
	o.ProviderConfig = vedro.ProviderConfig{}
	o.Logger.V(1).Info("getting ProviderConfig")

	o.Condition = metav1.Condition{
		Type: conditions.TypeProviderConfigReady,
	}

	err := o.KubeClient.Get(ctx, name, &o.ProviderConfig)

	if err != nil {
		o.Error = err
		o.Condition.Status = metav1.ConditionFalse

		if apierrors.IsNotFound(err) {
			o.Logger.Info("ProviderConfig not found")
			o.Condition.Reason = conditions.ReasonProviderConfigNotFound
			o.Condition.Message = "ProviderConfig was not found"
			return
		}
		o.Logger.Error(err, "failed to get ProviderConfig")

		o.Condition.Reason = conditions.ReasonProviderConfigGetFailed
		o.Condition.Message = err.Error()
		return
	}
}
