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
	"github.com/svetoch-dev/vedro/internal/capabilities"
	"github.com/svetoch-dev/vedro/internal/cloud/registry"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/resolvers"
)

const bucketFinalizer = "vedro.svetoch.dev/bucket-finalizer"

// BucketReconciler reconciles a Bucket object
type BucketReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Needed abstraction for tests
	ProviderFactory ProviderFactory
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=buckets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=buckets/finalizers,verbs=update
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs,verbs=create;update;get;list;watch

func (r *BucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	bucket := resolvers.BucketResolver{
		KubeClient: r.Client,
		Logger:     logger,
	}

	// Find bucket and set Conditions
	bucket.Resolve(ctx, req.NamespacedName)
	bucket.Condition.ObservedGeneration = bucket.Generation

	if !bucket.IsOk() {
		return ReconcileIgnoreNotFound(ctx, bucket.Error, "unable to fetch bucket")
	}

	logger = logger.WithValues(
		"bucketName", helpers.BucketNameFromCR(bucket.Bucket),
		"providerConfig", bucket.Spec.ProviderRef.Name,
	)

	ctx = log.IntoContext(ctx, logger)

	if result, err, handled := r.reconcileBucketFinalizer(ctx, req, &bucket); handled {
		return result, err
	}

	providerFactory := r.ProviderFactory
	if providerFactory == nil {
		providerFactory = registry.NewProvider
	}

	providerSetup, issue := prepareProvider(ctx, bucket.Spec.ProviderRef, r.Client, providerFactory)

	provider := providerSetup.Provider
	providerConfig := providerSetup.Config
	providerConfig.Condition.ObservedGeneration = bucket.Generation

	if provider != nil {
		defer func() {
			if err := provider.Cleanup(ctx); err != nil {
				logger.Error(err, "provider cleanup failed")
			}
		}()
	}

	if issue != nil {
		copyConditionState(&bucket.Condition, providerConfig.Condition)
		patchErr := r.patchStatus(ctx, req, bucket.Generation, func(p *vedro.Bucket) {
			meta.SetStatusCondition(&p.Status.Conditions, bucket.Condition)
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

	// check bucket capabilities
	caps := provider.Capabilities().Bucket
	unsupported := capabilities.ValidateBucketCapabilities(caps, bucket.Spec)
	bucket.Status.UnsupportedFeatures = unsupported

	if len(unsupported) > 0 {
		logger.Info("Bucket Unsupported features found")

		if bucket.Spec.UnsupportedFeaturePolicy == vedro.UnsupportedFeaturePolicyFail {
			logger.Info("UnsupportedFeaturePolicy set to Fail. stopping reconciliation")
			bucket.Condition.Status = metav1.ConditionFalse
			bucket.Condition.Reason = conditions.ReasonBucketUnsupportedFeatures
			bucket.Condition.Message = "unsupported features found"
			patchErr := r.patchStatus(ctx, req, bucket.Generation, func(b *vedro.Bucket) {
				b.Status.UnsupportedFeatures = bucket.Status.UnsupportedFeatures
				meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
			})
			if patchErr != nil {
				return ReconcileError(ctx, patchErr, "patch error")
			}

			return Reconciled()
		}
		if bucket.Spec.UnsupportedFeaturePolicy == vedro.UnsupportedFeaturePolicyWarn {
			patchErr := r.patchStatus(ctx, req, bucket.Generation, func(b *vedro.Bucket) {
				b.Status.UnsupportedFeatures = bucket.Status.UnsupportedFeatures
			})
			if patchErr != nil {
				return ReconcileError(ctx, patchErr, "patch error")
			}
		}

	}

	// check that spec is valid
	validationResult := provider.Bucket().ValidateBucketSpec(bucket.Bucket, providerConfig.Spec.Type)

	if !validationResult.Valid {
		logger.Info("spec is invalid")
		bucket.Condition.Status = metav1.ConditionFalse
		bucket.Condition.Reason = conditions.ReasonBucketInvalidSpec
		bucket.Condition.Message = validationResult.Message
		patchErr := r.patchStatus(ctx, req, bucket.Generation, func(b *vedro.Bucket) {
			meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
	}

	// Ensure that spec and bucket match
	result, err := provider.Bucket().EnsureBucket(ctx, bucket.Bucket)

	if err != nil {
		bucket.Condition.Status = metav1.ConditionFalse
		bucket.Condition.Reason = conditions.ReasonBucketEnsureError
		bucket.Condition.Message = err.Error()
		patchErr := r.patchStatus(ctx, req, bucket.Generation, func(b *vedro.Bucket) {
			meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}

		return ReconcileError(ctx, err, "EnsureBucket failed")
	}

	// Set bucket condition to reconciled and do a final patch
	bucket.Condition.Status = metav1.ConditionTrue
	bucket.Condition.Reason = conditions.ReasonBucketReconciled
	bucket.Condition.Message = "Bucket Reconciled"

	patchErr := r.patchStatus(ctx, req, bucket.Generation, func(b *vedro.Bucket) {
		b.Status.ExternalName = result.Name
		b.Status.ExternalId = result.Id
		b.Status.Location = result.Location
		b.Status.Applied = result.Properties
		b.Status.ObservedProvider = bucket.Spec.ProviderRef.Name
		b.Status.UnsupportedFeatures = bucket.Status.UnsupportedFeatures
		meta.SetStatusCondition(&b.Status.Conditions, providerConfig.Condition)
		meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
	})
	if patchErr != nil {
		return ReconcileError(ctx, patchErr, "patch error")
	}

	logger.Info("Bucket reconcile success")

	return Reconciled()
}

// reconcileBucketFinalizer adds the finalizer to active buckets and handles
// deletion paths
func (r *BucketReconciler) reconcileBucketFinalizer(
	ctx context.Context,
	req ctrl.Request,
	bucket *resolvers.BucketResolver,
) (ctrl.Result, error, bool) {
	if bucket.IsBeingDeleted() {
		result, err := r.deleteBucket(ctx, req, bucket)
		return result, err, true
	}

	if !controllerutil.ContainsFinalizer(&bucket.Bucket, bucketFinalizer) {
		controllerutil.AddFinalizer(&bucket.Bucket, bucketFinalizer)
		if err := r.Update(ctx, &bucket.Bucket); err != nil {
			result, reconcileErr := ReconcileError(ctx, err, "add finalizer error")
			return result, reconcileErr, true
		}
	}

	return ctrl.Result{}, nil, false
}

func (r *BucketReconciler) deleteBucket(
	ctx context.Context,
	req ctrl.Request,
	bucket *resolvers.BucketResolver,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(&bucket.Bucket, bucketFinalizer) {
		logger.Info("Bucket is being deleted, but finalizer is not set; skipping deletion handling")
		return Reconciled()
	}

	if bucket.Spec.DeletionPolicy == vedro.DeletionPolicyRetain {
		logger.Info("skipping Bucket deletion because deletionPolicy is Retain")
	}

	referenced, err := bucket.IsReferenced(ctx)

	if err != nil {
		return ReconcileError(ctx, err, "Unable to list BucketAccess objects")
	}

	if referenced {
		return ReconcileAfter(ctx, time.Second*10, "Bucket is referenced by BucketAccess objects waiting for them to be deleted. Requeuing after 10s")
	}

	if bucket.Spec.DeletionPolicy == vedro.DeletionPolicyDelete {
		logger.Info("deleting bucket and all of its objects")
		providerFactory := r.ProviderFactory
		if providerFactory == nil {
			providerFactory = registry.NewProvider
		}
		providerName := bucket.Status.ObservedProvider
		if providerName == "" {
			providerName = bucket.Spec.ProviderRef.Name
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
				"unable to prepare provider for Bucket deletion",
			)
		}

		err := provider.Bucket().DeleteBucket(ctx, bucket.Bucket)

		if err != nil {
			bucket.Condition.Status = metav1.ConditionFalse
			bucket.Condition.Reason = conditions.ReasonBucketDeleteError
			bucket.Condition.Message = err.Error()

			patchErr := r.patchStatus(ctx, req, bucket.Generation, func(b *vedro.Bucket) {
				meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
			})
			if patchErr != nil {
				return ReconcileError(ctx, patchErr, "patch error")
			}
			return ReconcileError(ctx, err, "unable to delete external bucket")
		}

	}
	controllerutil.RemoveFinalizer(&bucket.Bucket, bucketFinalizer)
	if err := r.Update(ctx, &bucket.Bucket); err != nil {
		return ReconcileError(ctx, err, "remove finalizer error")
	}

	return Reconciled()

}

func (r *BucketReconciler) patchStatus(
	ctx context.Context,
	req ctrl.Request,
	observedGeneration int64,
	mutate func(bucket *vedro.Bucket),
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var obj vedro.Bucket

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

func (r *BucketReconciler) findBucketsOfProviderConfig(
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

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      bucket.Name,
				Namespace: bucket.Namespace,
			},
		})
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *BucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&vedro.Bucket{},
		).
		Watches(
			// Watch ProviderConfig for changes and queue events for
			// buckets that reference it
			&vedro.ProviderConfig{},
			handler.EnqueueRequestsFromMapFunc(r.findBucketsOfProviderConfig),
		).
		Named("bucket").
		Complete(r)
}
