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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/cloud/registry"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/resolvers"
)

const principalFinalizer = "vedro.svetoch.dev/cloudprincipal-finalizer"

// BucketReconciler reconciles a Bucket object
type CloudPrincipalReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Needed abstraction for tests
	ProviderFactory func(
		ctx context.Context,
		cfg vedro.ProviderConfig,
		kubeClient client.Client,
	) (cloud.Provider, error)
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipals,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipals/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipals/finalizers,verbs=update
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs,verbs=create;update;get;list;watch

func (r *CloudPrincipalReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	principal := resolvers.CloudPrincipalResolver{
		KubeClient: r.Client,
		Logger:     logger,
	}

	// Find principal and set Conditions
	principal.Resolve(ctx, req.NamespacedName)
	principal.Condition.ObservedGeneration = principal.Generation

	if !principal.IsOk() {
		return ReconcileIgnoreNotFound(ctx, principal.Error, "unable to fetch CloudPrincipal")
	}

	logger = logger.WithValues(
		"principalName", helpers.PrincipalNameFromCR(principal.CloudPrincipal),
		"providerConfig", principal.Spec.ProviderRef.Name,
	)

	ctx = log.IntoContext(ctx, logger)

	if !controllerutil.ContainsFinalizer(&principal.CloudPrincipal, principalFinalizer) {
		controllerutil.AddFinalizer(&principal.CloudPrincipal, principalFinalizer)
		if err := r.Update(ctx, &principal.CloudPrincipal); err != nil {
			return ReconcileError(ctx, err, "add finalizer error")
		}
	}

	providerConfig := resolvers.ProviderConfigResolver{
		KubeClient: r.Client,
		Logger:     logger,
	}

	providerConfigName := types.NamespacedName{
		Name: principal.Spec.ProviderRef.Name,
	}

	// Find ProviderConfig and set condition
	providerConfig.Resolve(ctx, providerConfigName)
	providerConfig.Condition.ObservedGeneration = principal.Generation

	if !providerConfig.IsOk() {
		patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
			meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
		})

		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}

		return ReconcileIgnoreNotFound(ctx, providerConfig.Error, "unable to fetch ProviderConfig")
	}

	providerFactory := r.ProviderFactory
	if providerFactory == nil {
		providerFactory = registry.NewProvider
	}

	provider, err := providerFactory(ctx, providerConfig.ProviderConfig, r.Client)

	// If error change status conditions and end Reconcile
	if err != nil {
		logger.Error(err, "Error in setting NewProvider")
		providerConfig.Condition.Status = metav1.ConditionFalse
		providerConfig.Condition.Reason = conditions.ReasonProviderConfigError
		providerConfig.Condition.Message = err.Error()
		principal.Condition.Status = metav1.ConditionFalse
		principal.Condition.Reason = conditions.ReasonProviderConfigError
		principal.Condition.Message = err.Error()
		patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
			meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
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
		patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
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

	// when cr gets deleted metadata.deletionTimestamp
	// is set to current timestamp. If its not in process
	// of being deleted metadata.deletionTimestamp == 0
	if !principal.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&principal.CloudPrincipal, principalFinalizer) {
			logger.Info("CloudPrincipal is being deleted, but finalizer is not set; skipping deletion handling")
			return Reconciled()
		}

		if principal.Spec.DeletionPolicy == vedro.DeletionPolicyDelete {
			logger.Info("deleting CloudPrincipal")
			err := provider.Principal().DeletePrincipal(ctx, principal.CloudPrincipal)
			if err != nil {
				principal.Condition.Status = metav1.ConditionFalse
				principal.Condition.Reason = conditions.ReasonCloudPrincipalDeleteError
				principal.Condition.Message = err.Error()

				patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
					meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
				})
				if patchErr != nil {
					return ReconcileError(ctx, patchErr, "patch error")
				}
				return ReconcileErrorRAfter(ctx, err, time.Second*10, "unable to delete external CloudPrincipal")
			}
		} else {
			logger.Info("skipping CloudPrincipal deletion because deletionPolicy is Retain")
		}

		controllerutil.RemoveFinalizer(&principal.CloudPrincipal, principalFinalizer)
		if err := r.Update(ctx, &principal.CloudPrincipal); err != nil {
			return ReconcileError(ctx, err, "remove finalizer error")
		}
		return Reconciled()
	}

	// check that spec is valid
	validationResult := provider.Principal().ValidatePrincipalSpec(principal.CloudPrincipal)

	if !validationResult.Valid {
		logger.Info("spec is invalid")
		principal.Condition.Status = metav1.ConditionFalse
		principal.Condition.Reason = conditions.ReasonCloudPrincipalInvalidSpec
		principal.Condition.Message = validationResult.Message
		patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
	}

	// Ensure that spec and principal match
	result, err := provider.Principal().EnsurePrincipal(ctx, principal.CloudPrincipal)

	if err != nil {
		logger.Error(err, "EnsurePrincipal failed")
		principal.Condition.Status = metav1.ConditionFalse
		principal.Condition.Reason = conditions.ReasonCloudPrincipalEnsureError
		principal.Condition.Message = err.Error()
		patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
	}

	// Set principal condition to reconciled and do a final patch
	principal.Condition.Status = metav1.ConditionTrue
	principal.Condition.Reason = conditions.ReasonCloudPrincipalReconciled
	principal.Condition.Message = "CloudPrincipal Reconciled"

	patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
		p.Status.ExternalName = result.Name
		p.Status.ExternalId = result.Id
		p.Status.ObservedProvider = principal.Spec.ProviderRef.Name
		meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
		meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
	})
	if patchErr != nil {
		return ReconcileError(ctx, patchErr, "patch error")
	}

	logger.Info("CloudPrincipal reconcile success")

	return Reconciled()
}

func (r *CloudPrincipalReconciler) patchStatus(
	ctx context.Context,
	req ctrl.Request,
	observedGeneration int64,
	mutate func(principal *vedro.CloudPrincipal),
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var obj vedro.CloudPrincipal

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

func (r *CloudPrincipalReconciler) findCloudPrincipalsOfProviderConfig(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	providerConfig, ok := obj.(*vedro.ProviderConfig)
	if !ok {
		return nil
	}

	var list vedro.CloudPrincipalList
	if err := r.List(ctx, &list); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))

	for _, obj := range list.Items {
		if obj.Spec.ProviderRef.Name != providerConfig.Name {
			continue
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      obj.Name,
				Namespace: obj.Namespace,
			},
		})
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudPrincipalReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&vedro.CloudPrincipal{},
		).
		Watches(
			// Watch ProviderConfig for changes and queue events for
			// principals that reference it
			&vedro.ProviderConfig{},
			handler.EnqueueRequestsFromMapFunc(r.findCloudPrincipalsOfProviderConfig),
		).
		Named("CloudPrincipal").
		Complete(r)
}
