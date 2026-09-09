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
	"errors"
	"reflect"

	"github.com/svetoch-dev/vedro/internal/usagepolicy"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/svetoch-dev/vedro/internal/resolvers"
)

const principalAuthFinalizer = "vedro.svetoch.dev/cloudprincipalauth-finalizer"

// BucketReconciler reconciles a Bucket object
type CloudPrincipalAuthReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Needed abstraction for tests
	ProviderFactory ProviderFactory
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipals,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipals/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipals/finalizers,verbs=update
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs,verbs=create;update;get;list;watch
func (r *CloudPrincipalAuthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	principalAuth := resolvers.CloudPrincipalAuthResolver{
		KubeClient: r.Client,
		Logger:     logger,
	}

	// Find principalAuth and set Conditions
	principalAuth.Resolve(ctx, req.NamespacedName)
	principalAuth.Condition.ObservedGeneration = principalAuth.Generation

	if !principalAuth.IsOk() {
		return ReconcileIgnoreNotFound(ctx, principalAuth.Error, "unable to fetch CloudPrincipalAuth")
	}

	logger = logger.WithValues(
		"principalAuthName", principalAuth.Name,
		"principal", principalAuth.Spec.PrincipalRef.Name,
		"method", principalAuth.Spec.Method,
	)

	ctx = log.IntoContext(ctx, logger)

	if result, err, handled := r.reconcileCloudPrincipalFinalizer(ctx, req, &principalAuth); handled {
		return result, err
	}

	principal, principalStatus := prepareCloudPrincipal(ctx, r.Client, principalAuth.Spec.PrincipalRef)
	principal.Condition.ObservedGeneration = principalAuth.Generation

	switch principalStatus {
	case CloudPrincipalBeingDeleted:
		if err := r.Delete(ctx, &principalAuth.CloudPrincipalAuth); err != nil &&
			!apierrors.IsNotFound(err) {
			return ReconcileError(ctx, err, "delete CloudPrincipalAuth")
		}

		return Reconciled()

	case CloudPrincipalNotOk:
		copyConditionState(&principalAuth.Condition, principal.Condition)
		patchErr := r.patchStatus(ctx, req, principalAuth.Generation, func(p *vedro.CloudPrincipalAuth) {
			meta.SetStatusCondition(&p.Status.Conditions, principalAuth.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}

		return ReconcileIgnoreNotFound(
			ctx,
			principal.Error,
			"unable to fetch Principal",
		)

	case CloudPrincipalNotReady:
		principalAuth.Condition.Status = metav1.ConditionFalse
		principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthDependencyNotReady
		principalAuth.Condition.Message = "CloudPrincipal is not Ready"
	case CloudPrincipalOk:
		principal.Condition.Status = metav1.ConditionTrue
		principal.Condition.Reason = conditions.ReasonCloudPrincipalReady
		principal.Condition.Message = "CloudPrincipal Ready"
	}

	if principal.Condition.Status != metav1.ConditionTrue {
		patchErr := r.patchStatus(ctx, req, principalAuth.Generation, func(p *vedro.CloudPrincipalAuth) {
			meta.SetStatusCondition(&p.Status.Conditions, principalAuth.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
	}

	providerFactory := r.ProviderFactory
	if providerFactory == nil {
		providerFactory = registry.NewProvider
	}

	providerSetup, issue := prepareProvider(ctx, principal.Spec.ProviderRef, r.Client, providerFactory)

	provider := providerSetup.Provider
	providerConfig := providerSetup.Config
	providerConfig.Condition.ObservedGeneration = principalAuth.Generation

	if provider != nil {
		defer func() {
			if err := provider.Cleanup(ctx); err != nil {
				logger.Error(err, "provider cleanup failed")
			}
		}()
	}

	if issue != nil {
		copyConditionState(&principalAuth.Condition, providerConfig.Condition)
		patchErr := r.patchStatus(ctx, req, principalAuth.Generation, func(p *vedro.CloudPrincipalAuth) {
			meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principalAuth.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
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

	decision := usagepolicy.CheckPrincipalAuth(
		providerConfig.Spec.UsagePolicy,
		principalAuth.CloudPrincipalAuth,
	)

	if !decision.Allowed {
		logger.Info("spec is Restricted", "message", decision.Message)
		principalAuth.Condition.Status = metav1.ConditionFalse
		principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthSpecRestricted
		principalAuth.Condition.Message = decision.Message
		patchErr := r.patchStatus(ctx, req, principalAuth.Generation, func(p *vedro.CloudPrincipalAuth) {
			p.Status.UnsupportedFeatures = principalAuth.Status.UnsupportedFeatures
			meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principalAuth.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
	}

	return Reconciled()
}

// reconcileCloudPrincipalFinalizer adds the finalizer to active principal auths and
// handles deletion paths
func (r *CloudPrincipalAuthReconciler) reconcileCloudPrincipalFinalizer(
	ctx context.Context,
	req ctrl.Request,
	principalAuth *resolvers.CloudPrincipalAuthResolver,
) (ctrl.Result, error, bool) {
	if principalAuth.IsBeingDeleted() {
		result, err := r.deleteCloudPrincipalAuth(ctx, req, principalAuth)
		return result, err, true
	}

	if !controllerutil.ContainsFinalizer(&principalAuth.CloudPrincipalAuth, principalAuthFinalizer) {
		controllerutil.AddFinalizer(&principalAuth.CloudPrincipalAuth, principalAuthFinalizer)
		if err := r.Update(ctx, &principalAuth.CloudPrincipalAuth); err != nil {
			result, reconcileErr := ReconcileError(ctx, err, "add finalizer error")
			return result, reconcileErr, true
		}
	}

	return ctrl.Result{}, nil, false
}

func (r *CloudPrincipalAuthReconciler) deleteCloudPrincipalAuth(
	ctx context.Context,
	req ctrl.Request,
	principalAuth *resolvers.CloudPrincipalAuthResolver,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(&principalAuth.CloudPrincipalAuth, principalAuthFinalizer) {
		logger.Info("CloudPrincipalAuth is being deleted, but finalizer is not set; skipping deletion handling")
		return Reconciled()
	}

	if principalAuth.ShouldBeRetained() {
		logger.Info("skipping deletion of authentication material because deletionPolicy is Retain")
	}

	if principalAuth.ShouldBeDeleted() {
		if principalAuth.Status.Applied != nil {
			logger.Info("deleting auth material for CloudPrincipal")
			providerFactory := r.ProviderFactory
			if providerFactory == nil {
				providerFactory = registry.NewProvider
			}
			providerRef := vedro.ProviderConfigReference{
				Name: principalAuth.Status.ObservedProvider,
			}

			providerSetup, issue := prepareProvider(ctx, providerRef, r.Client, providerFactory)

			provider := providerSetup.Provider
			providerConfig := providerSetup.Config

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
					"unable to prepare provider for deletion of auth material of CloudPrincipal",
				)
			}

			// check usagePolicy
			decision := usagepolicy.CheckPrincipalAuth(
				providerConfig.Spec.UsagePolicy,
				principalAuth.CloudPrincipalAuth,
			)

			if decision.Allowed {
				err := errors.New("") //provider.Principal().DeletePrincipal(ctx, principal.CloudPrincipal)
				if err != nil {
					principalAuth.Condition.Status = metav1.ConditionFalse
					principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthDeleteError
					principalAuth.Condition.Message = err.Error()

					patchErr := r.patchStatus(ctx, req, principalAuth.Generation, func(p *vedro.CloudPrincipalAuth) {
						meta.SetStatusCondition(&p.Status.Conditions, principalAuth.Condition)
					})
					if patchErr != nil {
						return ReconcileError(ctx, patchErr, "patch error")
					}
					return ReconcileError(ctx, err, "unable to delete auth material for CloudPrincipal")
				}
			} else {
				logger.Info("skipping auth material deletion because CloudPrincipalAuth spec is restricted", "message", decision.Message)
				if principalAuth.IsProvisioned() {
					message := "Cant remove CloudPrincipalAuth because it was already provisioned" +
						" and spec is restricted by usagePolicy"
					logger.Info(message)
					principalAuth.Condition.Status = metav1.ConditionFalse
					principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthRemoveFinalizerError
					principalAuth.Condition.Message = message
					patchErr := r.patchStatus(ctx, req, principalAuth.Generation, func(p *vedro.CloudPrincipalAuth) {
						meta.SetStatusCondition(&p.Status.Conditions, principalAuth.Condition)
					})
					if patchErr != nil {
						return ReconcileError(ctx, patchErr, "patch error")
					}
					return Reconciled()
				}
			}
		}
	}

	controllerutil.RemoveFinalizer(&principalAuth.CloudPrincipalAuth, principalFinalizer)
	if err := r.Update(ctx, &principalAuth.CloudPrincipalAuth); err != nil {
		return ReconcileError(ctx, err, "remove finalizer error")
	}
	return Reconciled()

}

func (r *CloudPrincipalAuthReconciler) patchStatus(
	ctx context.Context,
	req ctrl.Request,
	observedGeneration int64,
	mutate func(principal *vedro.CloudPrincipalAuth),
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var obj vedro.CloudPrincipalAuth

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

func (r *CloudPrincipalAuthReconciler) findCloudPrincipalAuthsOfCloudPrincipal(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	principal, ok := obj.(*vedro.CloudPrincipal)
	if !ok {
		return nil
	}

	var list vedro.CloudPrincipalAuthList
	if err := r.List(ctx, &list); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list CloudPrincipalAuth objects")
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))

	for _, obj := range list.Items {
		if obj.Spec.PrincipalRef.Name != principal.Name ||
			obj.Spec.PrincipalRef.Namespace != principal.Namespace {
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

func (r *CloudPrincipalAuthReconciler) findCloudPrincipalAuthsOfProviderConfig(
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

		requests = append(requests, r.findCloudPrincipalAuthsOfCloudPrincipal(ctx, &obj)...)
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudPrincipalAuthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&vedro.CloudPrincipalAuth{},
		).
		Watches(
			// Watch CloudPrincipal for changes and queue events for
			// CloudPrincipalAuths that reference it
			&vedro.CloudPrincipal{},
			handler.EnqueueRequestsFromMapFunc(r.findCloudPrincipalAuthsOfCloudPrincipal),
		).
		Watches(
			// Watch ProviderConfig for changes and queue events for
			// CloudPrincipalAuths that reference it
			&vedro.ProviderConfig{},
			handler.EnqueueRequestsFromMapFunc(r.findCloudPrincipalAuthsOfProviderConfig),
		).
		Named("CloudPrincipalAuth").
		Complete(r)
}
