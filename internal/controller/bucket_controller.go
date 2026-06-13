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
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud/registry"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/resolvers"
	"github.com/svetoch-dev/vedro/internal/validation"
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
		if apierrors.IsNotFound(bucket.Error) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, bucket.Error
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
		patchErr := r.patchBucketStatus(ctx, req, func(b *vedrov1alpha1.Bucket) {
			meta.SetStatusCondition(&b.Status.Conditions, providerConfig.Condition)
		})

		if patchErr != nil {
			return ctrl.Result{}, patchErr
		}

		if apierrors.IsNotFound(providerConfig.Error) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, providerConfig.Error
	}

	provider, err := registry.NewProvider(ctx, providerConfig.ProviderConfig, r.Client)

	//If error change status conditions and end Reconcile
	if err != nil {
		providerConfig.Condition.Status = metav1.ConditionFalse
		providerConfig.Condition.Reason = conditions.ReasonProviderConfigError
		providerConfig.Condition.Message = err.Error()
		bucket.Condition.Status = metav1.ConditionFalse
		bucket.Condition.Reason = conditions.ReasonProviderConfigError
		bucket.Condition.Message = err.Error()
		patchErr := r.patchBucketStatus(ctx, req, func(b *vedrov1alpha1.Bucket) {
			meta.SetStatusCondition(&b.Status.Conditions, providerConfig.Condition)
			meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ctrl.Result{}, patchErr
		}
		return ctrl.Result{}, nil
	}

	//providerConfig is valid and povider is configured by now
	//set his final condition
	providerConfig.Condition.Status = metav1.ConditionTrue
	providerConfig.Condition.Reason = conditions.ReasonProviderConfigReconciled
	providerConfig.Condition.Message = "ProviderConfig Reconciled"

	//check bucket capabilities
	caps := provider.Capabilities().Bucket
	unsupported := validation.ValidateBucketCapabilities(caps, bucket.Spec)

	if len(unsupported) > 0 {
		bucket.Condition.Status = metav1.ConditionFalse
		bucket.Condition.Reason = conditions.ReasonBucketInvalidCapabilities
		bucket.Condition.Message = "Bucket invalid capabilities"
		patchErr := r.patchBucketStatus(ctx, req, func(b *vedrov1alpha1.Bucket) {
			b.Status.UnsupportedFeatures = unsupported
			meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
		})
		if patchErr != nil {
			return ctrl.Result{}, patchErr
		}

		if bucket.Spec.UnsupportedFeaturePolicy == vedrov1alpha1.UnsupportedFeaturePolicyFail {
			return ctrl.Result{}, nil
		}
	}

	//Set bucket condition to reconciled and do a final patch
	bucket.Condition.Status = metav1.ConditionTrue
	bucket.Condition.Reason = conditions.ReasonBucketReconciled
	bucket.Condition.Message = "Bucket Reconciled"

	patchErr := r.patchBucketStatus(ctx, req, func(b *vedrov1alpha1.Bucket) {
		b.Status.ObservedGeneration = bucket.Generation
		b.Status.ExternalName = bucket.Name
		b.Status.Location = bucket.Spec.Location
		b.Status.ObservedProvider = bucket.Spec.ProviderRef.Name
		meta.SetStatusCondition(&b.Status.Conditions, providerConfig.Condition)
		meta.SetStatusCondition(&b.Status.Conditions, bucket.Condition)
	})
	if patchErr != nil {
		return ctrl.Result{}, patchErr
	}

	log.Info(fmt.Sprintf("status success %q", providerConfig.Name))

	return ctrl.Result{}, nil
}

func (r *BucketReconciler) patchBucketStatus(
	ctx context.Context,
	req ctrl.Request,
	mutate func(bucket *vedrov1alpha1.Bucket),
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var bucket vedrov1alpha1.Bucket

		if err := r.Get(ctx, req.NamespacedName, &bucket); err != nil {
			return err
		}

		original := bucket.DeepCopy()

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
		For(&vedrov1alpha1.Bucket{}).
		Watches(
			&vedrov1alpha1.ProviderConfig{},
			handler.EnqueueRequestsFromMapFunc(r.findBucketsForProviderConfig),
		).
		Named("bucket").
		Complete(r)
}
