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

type ResourceResolver interface {
	Populate(ctx context.Context, name types.NamespacedName)
	SetStatusCondition(condition metav1.Condition)
	IsOk() bool
}

// ProviderConfig
type ProviderConfigResolver struct {
	Object vedrov1alpha1.ProviderConfig

	KubeClient client.Client
	Log        logr.Logger

	Condition metav1.Condition
	Error     error
}

func (p *ProviderConfigResolver) IsOk() bool {
	return p.Error == nil
}

func (p *ProviderConfigResolver) SetStatusCondition(condition metav1.Condition) {
	meta.SetStatusCondition(&p.Object.Status.Conditions, condition)
}

func (p *ProviderConfigResolver) Resolve(
	ctx context.Context,
	name types.NamespacedName,
) {
	p.Error = nil
	p.Object = vedrov1alpha1.ProviderConfig{}
	p.Log.V(1).Info("getting ProviderConfig", "name", name.String())

	p.Condition = metav1.Condition{
		Type: "ProviderConfigReady",
	}

	err := p.KubeClient.Get(ctx, name, &p.Object)
	if err != nil {
		p.Error = err
		p.Condition.Status = metav1.ConditionFalse

		if apierrors.IsNotFound(err) {
			p.Log.Info("ProviderConfig not found", "name", name.String())
			p.Condition.Reason = "ProviderConfigNotFound"
			p.Condition.Message = fmt.Sprintf("ProviderConfig %q was not found", name.Name)
			return
		}
		p.Log.Error(err, "failed to get ProviderConfig", "name", name.String())

		p.Condition.Reason = "ProviderConfigGetFailed"
		p.Condition.Message = err.Error()
		return
	}

	p.Condition.Status = metav1.ConditionTrue
	p.Condition.Reason = "ProviderConfigFound"
	p.Condition.Message = fmt.Sprintf("ProviderConfig %q was found", name.Name)
}
