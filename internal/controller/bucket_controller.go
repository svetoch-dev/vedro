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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/capabilities"
	"github.com/svetoch-dev/vedro/internal/cloud/registry"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/resolvers"
)

// BucketReconciler reconciles a Bucket object
type BucketReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=buckets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=buckets/finalizers,verbs=update
// +kubebuilder:rbac:groups=vedro.svetoch.dev,resources=providerconfigs,verbs=get;list;watch
func (r *BucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)
	bucket := resolvers.BucketResolver{
		KubeClient: r.Client,
		Log:        log,
	}

	//Find bucket and set Conditions
	bucket.Resolve(ctx, req.NamespacedName)
	bucket.Condition.ObservedGeneration = bucket.Generation

	if !bucket.IsOk() {
		return ReconcileIgnoreNotFound(ctx, bucket.Error, "unable to fetch bucket")
	}

	providerConfig := resolvers.ProviderConfigResolver{
		KubeClient: r.Client,
		Log:        log,
	}

	providerConfigName := types.NamespacedName{
		Name: bucket.Spec.ProviderRef.Name,
	}

	//Find ProviderConfig and set condition
	providerConfig.Resolve(ctx, providerConfigName)
	providerConfig.Condition.ObservedGeneration = bucket.Generation

	if !providerConfig.IsOk() {
		patchErr := r.patchBucketStatus(ctx, req, bucket.Generation, func(b *vedrov1alpha1.Bucket) {
			meta.SetStatusCondition(&b.Status.Conditions, providerConfig.Condition)
		})

		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}

		return ReconcileIgnoreNotFound(ctx, providerConfig.Error, "unable to fetch ProviderConfig")
	}

	provider, err := registry.NewProvider(ctx, providerConfig.ProviderConfig, r.Client)

	//If error change status conditions and end Reconcile
	if err != nil {
		log.Error(err, "Error in setting NewProvider", "ProviderConfig", providerConfig.Name)
		providerConfig.Condition.Status = metav1.ConditionFalse
		providerConfig.Condition.Reason = conditions.ReasonProviderConfigError
		providerConfig.Condition.Message = err.Error()
		bucket.Condition.Status = metav1.ConditionFalse
		bucket.Condition.Reason = conditions.ReasonProviderConfigError
		bucket.Condition.Message = err.Error()
		patchErr := r.patchBucketStatus(ctx, req, bucket.Generation, func(b *vedrov1alpha1.Bucket) {
			meta.SetStatusCondition(&b.Status.Conditions, providerConfig.Condition)
			meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
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

	//check bucket capabilities
	caps := provider.Capabilities().Bucket
	unsupported := capabilities.ValidateBucketCapabilities(caps, bucket.Spec)
	bucket.Status.UnsupportedFeatures = unsupported

	if len(unsupported) > 0 {
		log.Info("Unsupported features found")
		bucket.Condition.Status = metav1.ConditionFalse
		bucket.Condition.Reason = conditions.ReasonBucketUnsupportedFeatures
		bucket.Condition.Message = "unsupported features found"
		patchErr := r.patchBucketStatus(ctx, req, bucket.Generation, func(b *vedrov1alpha1.Bucket) {
			b.Status.UnsupportedFeatures = bucket.Status.UnsupportedFeatures
			meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}

		if bucket.Spec.UnsupportedFeaturePolicy == vedrov1alpha1.UnsupportedFeaturePolicyFail {
			log.Info("UnsupportedFeaturePolicy set to Fail. stopping reconciliation")
			return Reconciled()
		}
	}

	//check that spec is valid
	validationResult := provider.Bucket().ValidateBucketSpec(bucket.Bucket)

	if !validationResult.Valid {
		log.Info("spec is invalid")
		bucket.Condition.Status = metav1.ConditionFalse
		bucket.Condition.Reason = conditions.ReasonBucketInvalidSpec
		bucket.Condition.Message = validationResult.Message
		patchErr := r.patchBucketStatus(ctx, req, bucket.Generation, func(b *vedrov1alpha1.Bucket) {
			meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
	}

	//Ensure that spec and bucket match
	result, err := provider.Bucket().EnsureBucket(ctx, bucket.Bucket)

	if err != nil {
		log.Error(err, "EnsureBucket failed")
		bucket.Condition.Status = metav1.ConditionFalse
		bucket.Condition.Reason = conditions.ReasonBucketEnsureError
		bucket.Condition.Message = err.Error()
		patchErr := r.patchBucketStatus(ctx, req, bucket.Generation, func(b *vedrov1alpha1.Bucket) {
			meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ReconcileError(ctx, patchErr, "patch error")
		}
		return Reconciled()
	}

	//Set bucket condition to reconciled and do a final patch
	bucket.Condition.Status = metav1.ConditionTrue
	bucket.Condition.Reason = conditions.ReasonBucketReconciled
	bucket.Condition.Message = "Bucket Reconciled"

	patchErr := r.patchBucketStatus(ctx, req, bucket.Generation, func(b *vedrov1alpha1.Bucket) {
		b.Status.ExternalName = result.ExternalName
		b.Status.Location = result.Location
		b.Status.Applied = result.Applied
		b.Status.ObservedProvider = bucket.Spec.ProviderRef.Name
		b.Status.UnsupportedFeatures = bucket.Status.UnsupportedFeatures
		meta.SetStatusCondition(&b.Status.Conditions, providerConfig.Condition)
		meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
	})
	if patchErr != nil {
		return ReconcileError(ctx, patchErr, "patch error")
	}

	log.Info("reconciled success")

	return Reconciled()
}

func (r *BucketReconciler) patchBucketStatus(
	ctx context.Context,
	req ctrl.Request,
	observedGeneration int64,
	mutate func(bucket *vedrov1alpha1.Bucket),
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var bucket vedrov1alpha1.Bucket

		if err := r.Get(ctx, req.NamespacedName, &bucket); err != nil {
			return err
		}

		original := bucket.DeepCopy()

		bucket.Status.ObservedGeneration = observedGeneration
		mutate(&bucket)

		if reflect.DeepEqual(original.Status, bucket.Status) {
			return nil
		}

		return r.Status().Patch(ctx, &bucket, client.MergeFrom(original))
	})
}

func (r *BucketReconciler) findBucketsForProviderConfig(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	providerConfig, ok := obj.(*vedrov1alpha1.ProviderConfig)
	if !ok {
		return nil
	}

	var bucketList vedrov1alpha1.BucketList
	if err := r.List(ctx, &bucketList); err != nil {
		return nil
	}

	var requests []reconcile.Request

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
			&vedrov1alpha1.Bucket{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}), //Reconcile only when generation changes
		).
		Watches(
			//Watch ProviderConfig for changes and queue events for
			//buckets that reference it
			&vedrov1alpha1.ProviderConfig{},
			handler.EnqueueRequestsFromMapFunc(r.findBucketsForProviderConfig),
		).
		Named("bucket").
		Complete(r)
}
