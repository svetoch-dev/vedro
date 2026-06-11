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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
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

	bucket.Resolve(ctx, req.NamespacedName)

	if !bucket.IsOk() {
		if apierrors.IsNotFound(bucket.Error) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, bucket.Error
	}

	original := bucket.Object.DeepCopy()

	bucket.Object.Status.ObservedGeneration = original.Generation
	bucket.Object.Status.ExternalName = original.Name
	bucket.Object.Status.Location = original.Spec.Location
	bucket.Object.Status.ObservedProvider = original.Spec.ProviderRef.Name

	providerConfig := resolvers.ProviderConfigResolver{
		KubeClient: r.Client,
		Log:        log,
	}

	providerConfigName := types.NamespacedName{
		Name: bucket.Object.Spec.ProviderRef.Name,
	}

	providerConfig.Resolve(ctx, providerConfigName)

	providerConfig.Condition.ObservedGeneration = bucket.Object.Generation

	bucket.SetStatusCondition(providerConfig.Condition)

	if statusErr := r.Status().Patch(ctx, &bucket.Object, client.MergeFrom(original)); statusErr != nil {
		return ctrl.Result{}, statusErr
	}

	if !providerConfig.IsOk() {
		if apierrors.IsNotFound(providerConfig.Error) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, providerConfig.Error
	}
	//provider, err = registry.NewProvider(ctx, providerConfig, r.Client)

	log.Info(fmt.Sprintf("status success %q", providerConfig.Object.Name))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vedrov1alpha1.Bucket{}).
		Named("bucket").
		Complete(r)
}
