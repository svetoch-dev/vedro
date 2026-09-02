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
	ProviderFactory ProviderFactory
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

	if result, err, handled := r.reconcileCloudPrincipalFinalizer(ctx, req, &principal); handled {
		return result, err
	}

	providerFactory := r.ProviderFactory
	if providerFactory == nil {
		providerFactory = registry.NewProvider
	}

	providerSetup, issue := prepareProvider(ctx, principal.Spec.ProviderRef, r.Client, providerFactory)

	provider := providerSetup.Provider
	providerConfig := providerSetup.Config
	providerConfig.Condition.ObservedGeneration = principal.Generation

	if provider != nil {
		defer func() {
			if err := provider.Cleanup(ctx); err != nil {
				logger.Error(err, "provider cleanup failed")
			}
		}()
	}

	if issue != nil {
		copyConditionState(&principal.Condition, providerConfig.Condition)
		patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		switch issue.Kind {
		case ProviderResolveFailed:
			return ReconcileIgnoreNotFound(
				ctx,
				issue.Error,
				"unable to fetch ProviderConfig",
			)

		case ProviderSettingFailed:
			return ReconcileError(ctx, issue.Error, "Error in setting NewProvider")

		case ProviderConfigInvalid:
			return Reconciled()
		}
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
		principal.Condition.Status = metav1.ConditionFalse
		principal.Condition.Reason = conditions.ReasonCloudPrincipalEnsureError
		principal.Condition.Message = err.Error()
		patchErr := r.patchStatus(ctx, req, principal.Generation, func(p *vedro.CloudPrincipal) {
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return ReconcileError(ctx, err, "EnsurePrincipal failed")
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

// reconcileCloudPrincipalFinalizer adds the finalizer to active principals and
// handles deletion paths
func (r *CloudPrincipalReconciler) reconcileCloudPrincipalFinalizer(
	ctx context.Context,
	req ctrl.Request,
	principal *resolvers.CloudPrincipalResolver,
) (ctrl.Result, error, bool) {
	if principal.IsBeingDeleted() {
		result, err := r.deleteCloudPrincipal(ctx, req, principal)
		return result, err, true
	}

	if !controllerutil.ContainsFinalizer(&principal.CloudPrincipal, principalFinalizer) {
		controllerutil.AddFinalizer(&principal.CloudPrincipal, principalFinalizer)
		if err := r.Update(ctx, &principal.CloudPrincipal); err != nil {
			result, reconcileErr := ReconcileError(ctx, err, "add finalizer error")
			return result, reconcileErr, true
		}
	}

	return ctrl.Result{}, nil, false
}

func (r *CloudPrincipalReconciler) deleteCloudPrincipal(
	ctx context.Context,
	req ctrl.Request,
	principal *resolvers.CloudPrincipalResolver,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(&principal.CloudPrincipal, principalFinalizer) {
		logger.Info("CloudPrincipal is being deleted, but finalizer is not set; skipping deletion handling")
		return Reconciled()
	}

	if principal.ShouldBeRetained() {
		logger.Info("skipping CloudPrincipal deletion because deletionPolicy is Retain")
	}

	referenced, err := principal.IsReferenced(ctx)

	if err != nil {
		return ReconcileError(ctx, err, "Unable to list BucketAccess objects")
	}

	if referenced {
		return ReconcileAfter(ctx, time.Second*10, "CloudPrincipal is referenced by BucketAccess objects waiting for them to be deleted. Requeuing after 10s")
	}

	if principal.ShouldBeDeleted() {
		logger.Info("deleting CloudPrincipal")
		providerFactory := r.ProviderFactory
		if providerFactory == nil {
			providerFactory = registry.NewProvider
		}
		providerName := principal.Status.ObservedProvider
		if providerName == "" {
			providerName = principal.Spec.ProviderRef.Name
		}
		providerRef := vedro.ProviderConfigReference{
			Name: providerName,
		}

		providerSetup, issue := prepareProvider(ctx, providerRef, r.Client, providerFactory)

		provider := providerSetup.Provider

		if provider != nil {
			defer func() {
				if err := provider.Cleanup(ctx); err != nil {
					logger.Error(err, "provider cleanup failed")
				}
			}()
		}

		if issue != nil {
			return ReconcileError(
				ctx,
				issue.Error,
				"unable to prepare provider for CloudPrincipal deletion",
			)
		}

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
			return ReconcileError(ctx, err, "unable to delete external CloudPrincipal")
		}
	}

	controllerutil.RemoveFinalizer(&principal.CloudPrincipal, principalFinalizer)
	if err := r.Update(ctx, &principal.CloudPrincipal); err != nil {
		return ReconcileError(ctx, err, "remove finalizer error")
	}
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
		ctrl.LoggerFrom(ctx).Error(err, "unable to list CloudPrincipal objects")
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
