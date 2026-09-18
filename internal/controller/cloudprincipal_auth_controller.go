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

	"github.com/svetoch-dev/vedro/internal/capabilities"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/usagepolicy"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const (
	principalAuthFinalizer  = "vedro.svetoch.dev/cloudprincipalauth-finalizer"
	principalAuthFieldOwner = "vedro-cloudprincipal-auth"
)

// BucketReconciler reconciles a Bucket object
type CloudPrincipalAuthReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Needed abstraction for tests
	ProviderFactory ProviderFactory
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipalauths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipalauths/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=cloudprincipalauths/finalizers,verbs=update
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs,verbs=get;list;watch

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

	if principal.Spec.ManagementPolicy != vedro.PrincipalManagementPolicyManaged {
		principalAuth.Condition.Status = metav1.ConditionFalse
		principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalIsNotManaged
		principalAuth.Condition.Message = "CloudPrincipalAuth cant be used on a none managed CloudPrincipal"
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

	caps := provider.Capabilities().PrincipalAuth

	unsupported := capabilities.ValidatePrincipalAuthCapabilities(
		caps,
		principalAuth.Spec,
		principal.Spec.Kind,
	)
	principalAuth.Status.UnsupportedFeatures = unsupported

	if len(unsupported) > 0 {
		logger.Info("CloudPrincipalAuth Unsupported features found")
		principalAuth.Condition.Status = metav1.ConditionFalse
		principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthUnsupportedFeatures
		principalAuth.Condition.Message = "unsupported features found"
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

	result, err := provider.PrincipalAuth().EnsureAuthentication(
		ctx,
		principalAuth.CloudPrincipalAuth,
		principal.CloudPrincipal,
	)

	if err != nil {
		logger.Error(err, "EnsureAuthentication error")
		principalAuth.Condition.Status = metav1.ConditionFalse
		principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthEnsureError
		principalAuth.Condition.Message = err.Error()
		patchErr := r.patchStatus(ctx, req, principalAuth.Generation, func(p *vedro.CloudPrincipalAuth) {
			p.Status.UnsupportedFeatures = principalAuth.Status.UnsupportedFeatures
			meta.SetStatusCondition(&p.Status.Conditions, providerConfig.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principalAuth.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return ReconcileError(ctx, err, "EnsureAuthentication failed")
	}

	switch principalAuth.Spec.Method {
	case vedro.AuthMethodStaticCredentials:
		return r.ensureStaticCredentials(
			ctx,
			req,
			&principalAuth,
			&principal,
			&providerConfig,
			provider,
			result,
		)
	case vedro.AuthMethodWorkloadIdentity:
		return r.ensureWorkloadIdentity(
			ctx,
			req,
			&principalAuth,
			&principal,
			&providerConfig,
			result,
		)
	default:
		logger.Info("Method is not supported", "method", principalAuth.Spec.Method)
		return Reconciled()
	}
}

func (r *CloudPrincipalAuthReconciler) ensureWorkloadIdentity(
	ctx context.Context,
	req ctrl.Request,
	principalAuth *resolvers.CloudPrincipalAuthResolver,
	principal *resolvers.CloudPrincipalResolver,
	providerConfig *resolvers.ProviderConfigResolver,
	authResult *cloud.PrincipalAuthResult,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	patchWorkloadIdentityStatus := func(condition metav1.Condition) error {
		return r.patchStatus(ctx, req, principalAuth.Generation,
			func(p *vedro.CloudPrincipalAuth) {
				p.Status.UnsupportedFeatures = principalAuth.Status.UnsupportedFeatures
				p.Status.Applied = principalAuth.Status.Applied
				p.Status.ObservedProvider = principal.Spec.ProviderRef.Name

				for _, c := range []metav1.Condition{
					condition,
					providerConfig.Condition,
					principalAuth.Condition,
					principal.Condition,
				} {
					meta.SetStatusCondition(&p.Status.Conditions, c)
				}
			},
		)
	}

	if principalAuth.Status.Applied == nil {
		principalAuth.Status.Applied = &vedro.CloudPrincipalAuthProperties{
			Method:        principalAuth.Spec.Method,
			CredentialsId: authResult.CredentialsID,
			PrincipalId:   principal.Status.ExternalId,
		}
	}

	result, err, handled := r.reconcileServiceAccount(
		ctx,
		authResult.ServiceAccountPatch,
		principalAuth,
		patchWorkloadIdentityStatus,
	)

	if handled {
		if apierrors.IsNotFound(err) {
			return result, nil
		}
		return result, err
	}

	principalAuth.Condition.Status = metav1.ConditionTrue
	principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthReconciled
	principalAuth.Condition.Message = "CloudPrincipalAuth Reconciled"
	principalAuth.Status.Applied.ServiceAccountRef = &vedro.NamespacedName{
		Name:      authResult.ServiceAccountPatch.Name,
		Namespace: authResult.ServiceAccountPatch.Namespace,
	}
	configured := &metav1.Condition{
		Type:               conditions.TypeWorkloadIdentityConfigured,
		ObservedGeneration: principalAuth.Generation,
		Status:             metav1.ConditionTrue,
		Reason:             conditions.ReasonWorkloadIdentityReconciled,
		Message:            "WorkloadIdentity reconciled",
	}
	patchErr := patchWorkloadIdentityStatus(*configured)
	if patchErr != nil {
		return ReconcileError(ctx, patchErr, "patch error")
	}
	logger.Info("CloudPrincipalAuth reconcile success")
	return Reconciled()

}

func (r *CloudPrincipalAuthReconciler) reconcileServiceAccount(
	ctx context.Context,
	serviceAccount *corev1.ServiceAccount,
	principalAuth *resolvers.CloudPrincipalAuthResolver,
	patcher func(condition metav1.Condition) error,
) (ctrl.Result, error, bool) {

	configured := meta.FindStatusCondition(
		principalAuth.Status.Conditions,
		conditions.TypeWorkloadIdentityConfigured,
	)

	if configured == nil {
		configured = &metav1.Condition{
			Type:               conditions.TypeWorkloadIdentityConfigured,
			ObservedGeneration: principalAuth.Generation,
		}
	}

	key := types.NamespacedName{
		Namespace: serviceAccount.Namespace,
		Name:      serviceAccount.Name,
	}

	var sa corev1.ServiceAccount

	err := r.Get(
		ctx,
		key,
		&sa,
	)

	if err != nil {
		configured.Status = metav1.ConditionFalse
		configured.Reason = conditions.ReasonWorkloadIdentityError
		configured.Message = err.Error()
		copyConditionState(&principalAuth.Condition, *configured)
		patchErr := patcher(*configured)
		if patchErr != nil {
			res, rerr := ReconcileError(ctx, patchErr, "patch error")
			return res, rerr, true
		}

		if apierrors.IsNotFound(err) {
			res, rerr := ReconcileError(
				ctx,
				err,
				"ServiceAccount not found",
				"name", key.Name,
				"namespace", key.Namespace,
			)
			return res, rerr, true
		}

		res, rerr := ReconcileError(ctx, err, "error getting k8s service account")
		return res, rerr, true
	}

	// Use server side apply in order to easily delete any
	// fields set
	err = r.Patch(
		ctx,
		serviceAccount,
		client.Apply,
		client.FieldOwner(principalAuthFieldOwner),
	)

	if err != nil {
		configured.Status = metav1.ConditionFalse
		configured.Reason = conditions.ReasonWorkloadIdentityError
		configured.Message = err.Error()
		copyConditionState(&principalAuth.Condition, *configured)
		patchErr := patcher(*configured)
		if patchErr != nil {
			res, rerr := ReconcileError(ctx, patchErr, "patch error")
			return res, rerr, true
		}

		res, rerr := ReconcileError(ctx, err, "error patching k8s service account")
		return res, rerr, true
	}
	res, rerr := Reconciled()
	return res, rerr, false
}

func (r *CloudPrincipalAuthReconciler) ensureStaticCredentials(
	ctx context.Context,
	req ctrl.Request,
	principalAuth *resolvers.CloudPrincipalAuthResolver,
	principal *resolvers.CloudPrincipalResolver,
	providerConfig *resolvers.ProviderConfigResolver,
	provider cloud.Provider,
	authResult *cloud.PrincipalAuthResult,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	patchCredentialsStatus := func(condition metav1.Condition) error {
		return r.patchStatus(ctx, req, principalAuth.Generation,
			func(p *vedro.CloudPrincipalAuth) {
				p.Status.UnsupportedFeatures = principalAuth.Status.UnsupportedFeatures
				p.Status.Applied = principalAuth.Status.Applied
				p.Status.ObservedProvider = principal.Spec.ProviderRef.Name

				for _, c := range []metav1.Condition{
					condition,
					providerConfig.Condition,
					principalAuth.Condition,
					principal.Condition,
				} {
					meta.SetStatusCondition(&p.Status.Conditions, c)
				}
			},
		)
	}

	configured := meta.FindStatusCondition(
		principalAuth.Status.Conditions,
		conditions.TypeStaticCredentialsConfigured,
	)

	if configured == nil || len(authResult.SecretData) != 0 {
		principalAuth.Status.Applied = &vedro.CloudPrincipalAuthProperties{
			Method:        principalAuth.Spec.Method,
			CredentialsId: authResult.CredentialsID,
			PrincipalId:   principal.Status.ExternalId,
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      principalAuth.Spec.StaticCredentials.SecretRef.Name,
				Namespace: principalAuth.Namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: authResult.SecretData,
		}

		return r.reconcileSecret(
			ctx,
			secret,
			principalAuth,
			patchCredentialsStatus,
		)
	}

	configured.ObservedGeneration = principalAuth.Generation

	if principalAuth.Status.Applied.SecretRef == nil {
		deleteErr := provider.PrincipalAuth().DeleteAuthentication(ctx, principalAuth.CloudPrincipalAuth)
		if deleteErr != nil {
			principalAuth.Condition.Status = metav1.ConditionFalse
			principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthDeleteError
			principalAuth.Condition.Message = deleteErr.Error()
			patchErr := patchCredentialsStatus(*configured)
			if patchErr != nil {
				return ReconcileError(ctx, patchErr, "patch error")
			}
			return ReconcileError(ctx, deleteErr, "unable to delete auth material for CloudPrincipal")
		}
		return ReconcileAfter(ctx, time.Second*1, "No applied secret ref. Recreating cloud auth material")
	}

	var secret corev1.Secret

	err := r.Get(
		ctx,
		types.NamespacedName{
			Namespace: principalAuth.Namespace,
			Name:      principalAuth.Status.Applied.SecretRef.Name,
		},
		&secret,
	)

	if err != nil {
		if !apierrors.IsNotFound(err) {
			principalAuth.Condition.Status = metav1.ConditionFalse
			principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthError
			principalAuth.Condition.Message = err.Error()
			patchErr := patchCredentialsStatus(*configured)
			if patchErr != nil {
				return ReconcileError(ctx, patchErr, "patch error")
			}
			return ReconcileError(ctx, err, "Could not get Applied secret")
		}

		deleteErr := provider.PrincipalAuth().DeleteAuthentication(ctx, principalAuth.CloudPrincipalAuth)
		if deleteErr != nil {
			principalAuth.Condition.Status = metav1.ConditionFalse
			principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthDeleteError
			principalAuth.Condition.Message = deleteErr.Error()
			patchErr := patchCredentialsStatus(*configured)
			if patchErr != nil {
				return ReconcileError(ctx, patchErr, "patch error")
			}
			return ReconcileError(ctx, deleteErr, "unable to delete auth material for CloudPrincipal")
		}
		return ReconcileAfter(ctx, time.Second*1, "No secret found. Recreating cloud auth material")
	}

	if len(secret.Data) == 0 {
		deleteErr := provider.PrincipalAuth().DeleteAuthentication(ctx, principalAuth.CloudPrincipalAuth)
		if deleteErr != nil {
			principalAuth.Condition.Status = metav1.ConditionFalse
			principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthDeleteError
			principalAuth.Condition.Message = deleteErr.Error()
			patchErr := patchCredentialsStatus(*configured)
			if patchErr != nil {
				return ReconcileError(ctx, patchErr, "patch error")
			}
			return ReconcileError(ctx, deleteErr, "unable to delete auth material for CloudPrincipal")
		}
		return ReconcileAfter(ctx, time.Second*1, "No data in secret. Recreating cloud auth material")
	}

	principalAuth.Condition.Status = metav1.ConditionTrue
	principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthReconciled
	principalAuth.Condition.Message = "CloudPrincipalAuth Reconciled"

	patchErr := patchCredentialsStatus(*configured)
	if patchErr != nil {
		return ReconcileError(ctx, patchErr, "patch error")
	}
	logger.Info("CloudPrincipalAuth reconcile success")
	return Reconciled()

}

func (r *CloudPrincipalAuthReconciler) reconcileSecret(
	ctx context.Context,
	secret *corev1.Secret,
	principalAuth *resolvers.CloudPrincipalAuthResolver,
	patcher func(condition metav1.Condition) error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	staticCredsCondition := metav1.Condition{
		Type:               conditions.TypeStaticCredentialsConfigured,
		ObservedGeneration: principalAuth.Generation,
	}

	err := controllerutil.SetControllerReference(
		&principalAuth.CloudPrincipalAuth,
		secret,
		r.Scheme,
	)

	if err != nil {
		staticCredsCondition.Status = metav1.ConditionFalse
		staticCredsCondition.Reason = conditions.ReasonStaticCredentialsError
		staticCredsCondition.Message = err.Error()
		copyConditionState(&principalAuth.Condition, staticCredsCondition)
		patchErr := patcher(staticCredsCondition)
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}

		return ReconcileError(ctx, err, "k8s Secret set owner reference failed")
	}

	err = helpers.CreateOrUpdateOwned(
		ctx,
		r.Client,
		secret,
		&principalAuth.CloudPrincipalAuth,
	)

	if err != nil {
		staticCredsCondition.Status = metav1.ConditionFalse
		staticCredsCondition.Reason = conditions.ReasonStaticCredentialsError
		staticCredsCondition.Message = err.Error()
		copyConditionState(&principalAuth.Condition, staticCredsCondition)
		patchErr := patcher(staticCredsCondition)
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return ReconcileError(ctx, err, "k8s Secret creation error")
	}

	principalAuth.Status.Applied.SecretRef = &vedro.NamespacedName{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	principalAuth.Condition.Status = metav1.ConditionTrue
	principalAuth.Condition.Reason = conditions.ReasonCloudPrincipalAuthReconciled
	principalAuth.Condition.Message = "CloudPrincipalAuth Reconciled"
	staticCredsCondition.Status = metav1.ConditionTrue
	staticCredsCondition.Reason = conditions.ReasonStaticCredentialsReconciled
	staticCredsCondition.Message = "StaticCredentials created"

	patchErr := patcher(staticCredsCondition)
	if patchErr != nil {
		return ReconcileError(ctx, patchErr, "patch error")
	}

	logger.Info("CloudPrincipalAuth reconcile success")

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

	applied := principalAuth.Status.Applied

	if principalAuth.ShouldBeRetained() {
		logger.Info("skipping deletion of authentication material because deletionPolicy is Retain")
		if applied != nil {
			if applied.Method == vedro.AuthMethodStaticCredentials &&
				applied.SecretRef != nil {
				err := helpers.RemoveAllOwnerRefs(
					ctx,
					r.Client,
					types.NamespacedName{
						Name:      applied.SecretRef.Name,
						Namespace: applied.SecretRef.Namespace,
					},
					&corev1.Secret{},
				)

				if err != nil {
					return ReconcileError(
						ctx,
						err,
						"Error removing owner ref for secret owned by CloudPrincipalAuth",
						"name", applied.SecretRef.Name,
						"namespace", applied.SecretRef.Namespace,
					)
				}
			}
		}

	}

	if principalAuth.ShouldBeDeleted() {
		if applied != nil {
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
				err := provider.PrincipalAuth().DeleteAuthentication(
					ctx,
					principalAuth.CloudPrincipalAuth,
				)
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

				if applied.Method == vedro.AuthMethodWorkloadIdentity &&
					applied.ServiceAccountRef != nil {
					sa := &corev1.ServiceAccount{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "ServiceAccount",
						},
						ObjectMeta: v1.ObjectMeta{
							Name:      applied.ServiceAccountRef.Name,
							Namespace: applied.ServiceAccountRef.Namespace,
						},
					}

					patcher := func(condition metav1.Condition) error {
						return r.patchStatus(ctx, req, principalAuth.Generation,
							func(p *vedro.CloudPrincipalAuth) {
								for _, c := range []metav1.Condition{
									condition,
									principalAuth.Condition,
								} {
									meta.SetStatusCondition(&p.Status.Conditions, c)
								}
							},
						)
					}

					result, err, handled := r.reconcileServiceAccount(
						ctx,
						sa,
						principalAuth,
						patcher,
					)

					if handled && !apierrors.IsNotFound(err) {
						return result, err
					}
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

	controllerutil.RemoveFinalizer(&principalAuth.CloudPrincipalAuth, principalAuthFinalizer)
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

func (r *CloudPrincipalAuthReconciler) findCloudPrincipalAuthsOfServiceAccount(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	sa, ok := obj.(*corev1.ServiceAccount)
	if !ok {
		return nil
	}

	var list vedro.CloudPrincipalAuthList
	if err := r.List(
		ctx,
		&list,
		client.InNamespace(sa.Namespace),
		client.MatchingFields{
			"spec.workloadIdentity.serviceAccountRef": sa.Name,
		},
	); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list CloudPrincipalAuth objects")
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))

	for _, obj := range list.Items {
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
	mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&vedro.CloudPrincipalAuth{},
		"spec.workloadIdentity.serviceAccountRef",
		func(obj client.Object) []string {
			auth := obj.(*vedro.CloudPrincipalAuth)

			if auth.Spec.WorkloadIdentity == nil ||
				auth.Spec.WorkloadIdentity.ServiceAccountRef.Name == "" {
				return nil
			}

			return []string{auth.Spec.WorkloadIdentity.ServiceAccountRef.Name}
		},
	)
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
		Watches(
			&corev1.ServiceAccount{},
			handler.EnqueueRequestsFromMapFunc(r.findCloudPrincipalAuthsOfServiceAccount),
		).
		Owns(&corev1.Secret{}).
		Named("CloudPrincipalAuth").
		Complete(r)
}
