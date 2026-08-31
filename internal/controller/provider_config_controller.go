/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud/registry"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/resolvers"
)

const providerConfigFinalizer = "vedro.svetoch.dev/providerconfig-finalizer"

type ProviderConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Needed abstraction for tests
	ProviderFactory ProviderFactory
}

// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs/finalizers,verbs=update

func (r *ProviderConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	providerConfig := resolvers.ProviderConfigResolver{
		KubeClient: r.Client,
		Logger:     logger,
	}

	// Find providerConfig and set Conditions
	providerConfig.Resolve(ctx, req.NamespacedName)
	providerConfig.Condition.ObservedGeneration = providerConfig.Generation

	if !providerConfig.IsOk() {
		return ReconcileIgnoreNotFound(ctx, providerConfig.Error, "unable to fetch ProviderConfig")
	}

	logger = logger.WithValues(
		"providerConfig", providerConfig.Name,
	)

	ctx = log.IntoContext(ctx, logger)

	if result, err, handled := r.reconcileProviderConfigFinalizer(ctx, req, &providerConfig); handled {
		return result, err
	}

	providerFactory := r.ProviderFactory
	if providerFactory == nil {
		providerFactory = registry.NewProvider
	}

	provider, err := providerFactory(ctx, providerConfig.ProviderConfig, r.Client)
	if err != nil {
		providerConfig.Condition.Status = metav1.ConditionFalse
		providerConfig.Condition.Reason = conditions.ReasonProviderConfigError
		providerConfig.Condition.Message = err.Error()
		patchErr := r.patchStatus(ctx, req, providerConfig.Generation, func(p *vedro.ProviderConfig) {
			meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return ReconcileError(ctx, err, "ProviderConfig error")
	}

	defer func() {
		if err := provider.Cleanup(ctx); err != nil {
			logger.Error(err, "provider cleanup failed")
		}
	}()

	validationResultCfg := provider.ValidateProviderConfigSpec(providerConfig.ProviderConfig)
	if !validationResultCfg.Valid {
		logger.Info("ProviderConfig.spec is invalid", "reason", validationResultCfg.Message)
		providerConfig.Condition.Status = metav1.ConditionFalse
		providerConfig.Condition.Reason = conditions.ReasonProviderConfigInvalidSpec
		providerConfig.Condition.Message = validationResultCfg.Message
		patchErr := r.patchStatus(ctx, req, providerConfig.Generation, func(p *vedro.ProviderConfig) {
			meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
	}

	// providerConfig is valid and provider is configured by now;
	// set its final condition.
	providerConfig.Condition.Status = metav1.ConditionTrue
	providerConfig.Condition.Reason = conditions.ReasonProviderConfigReconciled
	providerConfig.Condition.Message = "ProviderConfig Reconciled"
	patchErr := r.patchStatus(ctx, req, providerConfig.Generation, func(p *vedro.ProviderConfig) {
		meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
	})
	if patchErr != nil {
		return ReconcileError(ctx, patchErr, "patch error")
	}

	logger.Info("ProviderConfig reconcile success")

	return Reconciled()
}

// reconcileProviderConfigFinalizer adds the finalizer to active principals and
// handles deletion paths
func (r *ProviderConfigReconciler) reconcileProviderConfigFinalizer(
	ctx context.Context,
	req ctrl.Request,
	providerConfig *resolvers.ProviderConfigResolver,
) (ctrl.Result, error, bool) {
	if providerConfig.IsBeingDeleted() {
		result, err := r.deleteProviderConfig(ctx, req, providerConfig)
		return result, err, true
	}

	if !controllerutil.ContainsFinalizer(&providerConfig.ProviderConfig, providerConfigFinalizer) {
		controllerutil.AddFinalizer(&providerConfig.ProviderConfig, providerConfigFinalizer)
		if err := r.Update(ctx, &providerConfig.ProviderConfig); err != nil {
			result, reconcileErr := ReconcileError(ctx, err, "add finalizer error")
			return result, reconcileErr, true
		}
	}

	return ctrl.Result{}, nil, false
}

func (r *ProviderConfigReconciler) deleteProviderConfig(
	ctx context.Context,
	req ctrl.Request,
	providerConfig *resolvers.ProviderConfigResolver,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(&providerConfig.ProviderConfig, providerConfigFinalizer) {
		logger.Info("ProviderConfig is being deleted, but finalizer is not set; skipping deletion handling")
		return Reconciled()
	}

	referenced, err := providerConfig.IsReferenced(ctx)

	if err != nil {
		return ReconcileError(ctx, err, "Unable to list Bucket, CloudPrincipal objects")
	}

	if referenced {
		return ReconcileAfter(ctx, time.Second*10, "ProviderConfig is referenced by CustomResources waiting for them to be deleted. Requeuing after 10s")
	}

	controllerutil.RemoveFinalizer(&providerConfig.ProviderConfig, providerConfigFinalizer)
	if err := r.Update(ctx, &providerConfig.ProviderConfig); err != nil {
		return ReconcileError(ctx, err, "remove finalizer error")
	}

	return Reconciled()

}

func (r *ProviderConfigReconciler) patchStatus(
	ctx context.Context,
	req ctrl.Request,
	observedGeneration int64,
	mutate func(providerConfig *vedro.ProviderConfig),
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var obj vedro.ProviderConfig

		if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
			return err
		}

		original := obj.DeepCopy()

		obj.Status.ObservedGeneration = observedGeneration
		mutate(&obj)

		if reflect.DeepEqual(original.Status, obj.Status) {
			return nil
		}

		return r.Status().Patch(ctx, &obj, client.MergeFrom(original))
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProviderConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&vedro.ProviderConfig{},
		).
		Named("ProviderConfig").
		Complete(r)
}
