package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/conditions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createProviderConfig(ctx context.Context) {
	createProviderConfigNamed(ctx, "test-provider")
}

func createProviderConfigNamed(ctx context.Context, name string) {
	providerConfig := &vedro.ProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vedro.svetoch.dev/v1alpha1",
			Kind:       "ProviderConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: vedro.ProviderConfigSpec{
			Type:      vedro.ProviderTypeGCP,
			ProjectId: "test-project",
			Region:    "europe-west1",
			Method:    vedro.AuthMethodWorkloadIdentity,
		},
	}

	Expect(k8sClient.Create(ctx, providerConfig)).To(Succeed())

	markProviderConfigReady(ctx, providerConfig)

	DeferCleanup(func() {
		cleanupProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
	})

}

func getProviderConfig(ctx context.Context, key client.ObjectKey) *vedro.ProviderConfig {
	fetched := &vedro.ProviderConfig{}
	Expect(k8sClient.Get(
		ctx,
		key,
		fetched,
	)).To(Succeed())

	return fetched
}

func markProviderConfigReady(ctx context.Context, providerConfig *vedro.ProviderConfig) {
	fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))

	fetched.Status.ObservedGeneration = fetched.Generation

	meta.SetStatusCondition(&fetched.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: fetched.Generation,
		Reason:             conditions.ReasonProviderConfigReconciled,
		Message:            "ProviderConfig Reconciled",
	})

	Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
}

func markProviderConfigNotReady(ctx context.Context, providerConfig *vedro.ProviderConfig) {
	fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))

	fetched.Status.ObservedGeneration = fetched.Generation

	meta.SetStatusCondition(&fetched.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: fetched.Generation,
		Reason:             conditions.ReasonProviderConfigError,
		Message:            "ProviderConfig Error",
	})

	Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
}

func cleanupProviderConfig(ctx context.Context, key client.ObjectKey) {
	providerConfig := &vedro.ProviderConfig{}
	err := k8sClient.Get(ctx, key, providerConfig)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Delete(ctx, providerConfig)
	if err != nil {
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}
}
