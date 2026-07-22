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
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/resolvers"
)

const bucketAccessFinalizer = "vedro.svetoch.dev/bucketaccess-finalizer"

// BucketAccessReconciler reconciles a Bucket object
type BucketAccessReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Needed abstraction for tests
	ProviderFactory ProviderFactory
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=bucketaccesses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=bucketaccesses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=bucketaccesses/finalizers,verbs=update
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs,verbs=create;update;get;list;watch

func (r *BucketAccessReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	bucketAccess := resolvers.BucketAccessResolver{
		KubeClient: r.Client,
		Logger:     logger,
	}

	// Find bucketAccess and set Conditions
	bucketAccess.Resolve(ctx, req.NamespacedName)
	bucketAccess.Condition.ObservedGeneration = bucketAccess.Generation

	if !bucketAccess.IsOk() {
		return ReconcileIgnoreNotFound(ctx, bucketAccess.Error, "unable to fetch bucketAccess")
	}

	logger = logger.WithValues(
		"bucketAccessName", bucketAccess.Name,
		"bucket", bucketAccess.Spec.BucketRef.Name,
		"principal", bucketAccess.Spec.PrincipalRef.Name,
	)

	ctx = log.IntoContext(ctx, logger)

	if !controllerutil.ContainsFinalizer(&bucketAccess.BucketAccess, bucketAccessFinalizer) {
		controllerutil.AddFinalizer(&bucketAccess.BucketAccess, bucketAccessFinalizer)
		if err := r.Update(ctx, &bucketAccess.BucketAccess); err != nil {
			return ReconcileError(ctx, err, "add finalizer error")
		}
	}

	bucket := resolvers.BucketResolver{
		KubeClient: r.Client,
		Logger:     logger,
	}

	bucket.Resolve(ctx, types.NamespacedName{
		Namespace: bucketAccess.Spec.BucketRef.Namespace,
		Name:      bucketAccess.Spec.BucketRef.Name,
	})
	bucket.Condition.ObservedGeneration = bucketAccess.Generation

	if !bucket.IsOk() {
		bucketAccess.Condition.Status = bucket.Condition.Status
		bucketAccess.Condition.Reason = bucket.Condition.Reason
		bucketAccess.Condition.Message = bucket.Condition.Message
		patchErr := r.patchStatus(ctx, req, bucketAccess.Generation, func(p *vedro.BucketAccess) {
			meta.SetStatusCondition(&p.Status.Conditions, bucketAccess.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}

		return ReconcileIgnoreNotFound(
			ctx,
			bucket.Error,
			"unable to fetch Bucket",
		)
	}

	principal := resolvers.CloudPrincipalResolver{
		KubeClient: r.Client,
		Logger:     logger,
	}

	principal.Resolve(ctx, types.NamespacedName{
		Namespace: bucketAccess.Spec.PrincipalRef.Namespace,
		Name:      bucketAccess.Spec.PrincipalRef.Name,
	})
	principal.Condition.ObservedGeneration = bucketAccess.Generation

	if !principal.IsOk() {
		bucketAccess.Condition.Status = principal.Condition.Status
		bucketAccess.Condition.Reason = principal.Condition.Reason
		bucketAccess.Condition.Message = principal.Condition.Message
		patchErr := r.patchStatus(ctx, req, bucketAccess.Generation, func(p *vedro.BucketAccess) {
			meta.SetStatusCondition(&p.Status.Conditions, bucketAccess.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
			meta.SetStatusCondition(&p.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}

		return ReconcileIgnoreNotFound(
			ctx,
			principal.Error,
			"unable to fetch Principal",
		)
	}

	// Set bucketAccess condition to reconciled and do a final patch
	bucketAccess.Condition.Status = metav1.ConditionTrue
	bucketAccess.Condition.Reason = conditions.ReasonBucketAccessReconciled
	bucketAccess.Condition.Message = "BucketAccess Reconciled"
	patchErr := r.patchStatus(ctx, req, bucketAccess.Generation, func(p *vedro.BucketAccess) {
		meta.SetStatusCondition(&p.Status.Conditions, bucketAccess.Condition)
		meta.SetStatusCondition(&p.Status.Conditions, principal.Condition)
		meta.SetStatusCondition(&p.Status.Conditions, bucket.Condition)
	})
	if patchErr != nil {
		return ReconcileError(ctx, patchErr, "patch error")
	}

	logger.Info("BucketAccess reconcile success")

	return Reconciled()
}

func (r *BucketAccessReconciler) patchStatus(
	ctx context.Context,
	req ctrl.Request,
	observedGeneration int64,
	mutate func(bucketAccess *vedro.BucketAccess),
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var obj vedro.BucketAccess

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

func (r *BucketAccessReconciler) findBucketAccessOfBucket(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	bucket, ok := obj.(*vedro.Bucket)
	if !ok {
		return nil
	}

	var bucketAccessList vedro.BucketAccessList
	if err := r.List(ctx, &bucketAccessList); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list BucketAccess objects")
		return nil
	}

	requests := make([]reconcile.Request, 0, len(bucketAccessList.Items))

	for _, bucketAccess := range bucketAccessList.Items {
		if bucketAccess.Spec.BucketRef.Namespace != bucket.Namespace ||
			bucketAccess.Spec.BucketRef.Name != bucket.Name {
			continue
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      bucketAccess.Name,
				Namespace: bucketAccess.Namespace,
			},
		})
	}

	return requests
}

func (r *BucketAccessReconciler) findBucketAccessOfPrincipal(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	principal, ok := obj.(*vedro.CloudPrincipal)
	if !ok {
		return nil
	}

	var bucketAccessList vedro.BucketAccessList
	if err := r.List(ctx, &bucketAccessList); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list BucketAccess objects")
		return nil
	}

	requests := make([]reconcile.Request, 0, len(bucketAccessList.Items))

	for _, bucketAccess := range bucketAccessList.Items {
		if bucketAccess.Spec.PrincipalRef.Namespace != principal.Namespace ||
			bucketAccess.Spec.PrincipalRef.Name != principal.Name {
			continue
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      bucketAccess.Name,
				Namespace: bucketAccess.Namespace,
			},
		})
	}

	return requests
}

func (r *BucketAccessReconciler) findBucketAccessOfProviderConfig(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	providerConfig, ok := obj.(*vedro.ProviderConfig)
	if !ok {
		return nil
	}

	var bucketList vedro.BucketList
	if err := r.List(ctx, &bucketList); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to list Bucket objects")
		return nil
	}

	requests := make([]reconcile.Request, 0, len(bucketList.Items))

	for _, bucket := range bucketList.Items {
		if bucket.Spec.ProviderRef.Name != providerConfig.Name {
			continue
		}

		requests = append(requests, r.findBucketAccessOfBucket(ctx, &bucket)...)
	}

	return requests

}

// SetupWithManager sets up the controller with the Manager.
func (r *BucketAccessReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&vedro.BucketAccess{},
		).
		Watches(
			// Watch Bucket for changes and queue events for
			// bucketAccesss that reference it
			&vedro.Bucket{},
			handler.EnqueueRequestsFromMapFunc(r.findBucketAccessOfBucket),
		).
		Watches(
			// Watch CloudPrincipal for changes and queue events for
			// bucketAccesss that reference it
			&vedro.CloudPrincipal{},
			handler.EnqueueRequestsFromMapFunc(r.findBucketAccessOfPrincipal),
		).
		Watches(
			// Watch ProviderConfig for changes and queue events for
			// bucketAccesss that reference it
			&vedro.ProviderConfig{},
			handler.EnqueueRequestsFromMapFunc(r.findBucketAccessOfProviderConfig),
		).
		Named("bucketAccess").
		Complete(r)
}
