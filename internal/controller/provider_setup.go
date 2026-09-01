package controller

import (
	"context"
	"errors"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/resolvers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ProviderFactory func(
	context.Context,
	vedro.ProviderConfig,
	client.Client,
) (cloud.Provider, error)

type ProviderSetup struct {
	Provider cloud.Provider
	Config   resolvers.ProviderConfigResolver
}

type ProviderSetupIssueKind int

const (
	ProviderResolveFailed ProviderSetupIssueKind = 0
	ProviderSettingFailed ProviderSetupIssueKind = 1
	ProviderConfigInvalid ProviderSetupIssueKind = 2
)

type ProviderSetupIssue struct {
	Kind  ProviderSetupIssueKind
	Error error
}

func prepareProvider(
	ctx context.Context,
	config vedro.ProviderConfigReference,
	kubeClient client.Client,
	factory ProviderFactory,
) (ProviderSetup, *ProviderSetupIssue) {
	logger := log.FromContext(ctx)
	providerConfig := resolvers.ProviderConfigResolver{
		KubeClient: kubeClient,
		Logger:     logger,
	}

	providerConfigName := types.NamespacedName{
		Name: config.Name,
	}
	providerConfig.Resolve(ctx, providerConfigName)

	providerSetup := ProviderSetup{
		Config: providerConfig,
	}

	if !providerConfig.IsOk() {
		issue := ProviderSetupIssue{
			Kind:  ProviderResolveFailed,
			Error: providerConfig.Error,
		}
		return providerSetup, &issue
	}

	_, ok := providerConfig.IsReady()

	if !ok && !providerConfig.IsBeingDeleted() {
		msg := "ProviderConfig is not Ready"
		logger.Info(msg)
		providerSetup.Config.Condition.Status = metav1.ConditionFalse
		providerSetup.Config.Condition.Reason = conditions.ReasonProviderConfigNotReady
		providerSetup.Config.Condition.Message = msg
		issue := ProviderSetupIssue{
			Kind:  ProviderConfigInvalid,
			Error: errors.New(msg),
		}
		return providerSetup, &issue

	}

	provider, err := factory(ctx, providerConfig.ProviderConfig, kubeClient)
	if err != nil {
		providerSetup.Config.Condition.Status = metav1.ConditionFalse
		providerSetup.Config.Condition.Reason = conditions.ReasonProviderConfigError
		providerSetup.Config.Condition.Message = err.Error()
		issue := ProviderSetupIssue{
			Kind:  ProviderSettingFailed,
			Error: err,
		}
		return providerSetup, &issue
	}

	providerSetup.Provider = provider

	// providerConfig is valid and provider is configured by now;
	// set its final condition.
	providerSetup.Config.Condition.Status = metav1.ConditionTrue
	providerSetup.Config.Condition.Reason = conditions.ReasonProviderConfigReconciled
	providerSetup.Config.Condition.Message = "ProviderConfig Reconciled"

	return providerSetup, nil
}
