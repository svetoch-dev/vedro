package controller

import (
	"context"
	"fmt"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ensureProviderConfig(
	ctx context.Context,
	ref vedrov1alpha1.ProviderConfigReference,
	kubeClient client.Client,
) (vedrov1alpha1.ProviderConfig, metav1.Condition, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("getting ProviderConfig")

	var providerConfig vedrov1alpha1.ProviderConfig

	condition := metav1.Condition{
		Type: "ProviderConfigReady",
	}

	err := kubeClient.Get(ctx, types.NamespacedName{
		Name: ref.Name,
	}, &providerConfig)

	if err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("ProviderConfig not found")
			condition.Status = metav1.ConditionFalse
			condition.Reason = "ProviderConfigNotFound"
			condition.Message = fmt.Sprintf("ProviderConfig %q was not found", ref.Name)
		} else {
			condition.Status = metav1.ConditionFalse
			condition.Reason = "ProviderConfigError"
			condition.Message = err.Error()
		}
		return providerConfig, condition, err
	}
	condition.Status = metav1.ConditionTrue
	condition.Reason = "ProviderConfigfound"
	condition.Message = fmt.Sprintf("ProviderConfig %q was found", ref.Name)

	return providerConfig, condition, err
}
