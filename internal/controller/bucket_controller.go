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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
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

	var bucket vedrov1alpha1.Bucket
	err := r.Get(ctx, req.NamespacedName, &bucket)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	bucket.Status.ObservedGeneration = bucket.Generation
	bucket.Status.ExternalName = bucket.Name
	bucket.Status.Location = bucket.Spec.Location
	bucket.Status.ObservedProvider = bucket.Spec.ProviderRef.Name

	providerConfig, providerCondition, err := ensureProviderConfig(
		ctx,
		bucket.Spec.ProviderRef,
		r.Client,
	)
	providerCondition.ObservedGeneration = bucket.Generation
	meta.SetStatusCondition(&bucket.Status.Conditions, providerCondition)

	if err != nil {
		if apierrors.IsNotFound(err) {
			err = r.Status().Update(ctx, &bucket)
			if err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	err = r.Status().Update(ctx, &bucket)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info(fmt.Sprintf("status success %q", providerConfig.Name))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vedrov1alpha1.Bucket{}).
		Named("bucket").
		Complete(r)
}
