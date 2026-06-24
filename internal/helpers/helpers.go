package helpers

import (
	"context"
	"fmt"
	"maps"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func BucketNameFromCR(bckt vedrov1alpha1.Bucket) string {
	bucketName := bckt.Name

	if bckt.Spec.Name != "" {
		bucketName = bckt.Spec.Name
	}

	return bucketName
}

func GetSecretData(
	ctx context.Context,
	kubeClient client.Client,
	secretRef corev1.SecretReference,
	keys ...string,
) (map[string][]byte, error) {
	var secret corev1.Secret

	err := kubeClient.Get(ctx, client.ObjectKey{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}, &secret)
	if err != nil {
		return nil, fmt.Errorf("get secret %s/%s data failed: %w",
			secretRef.Namespace,
			secretRef.Name,
			err,
		)
	}

	data := make(map[string][]byte, len(keys))

	for _, key := range keys {
		value, ok := secret.Data[key]
		if !ok {
			return nil, fmt.Errorf("secret %s/%s does not contain key %q",
				secretRef.Namespace,
				secretRef.Name,
				key,
			)
		}

		data[key] = value
	}

	return data, nil
}

func AppliedState(
	location string,
	bckt vedrov1alpha1.Bucket,
) *cloud.BucketAttrs {
	spec := bckt.Spec
	bucketName := BucketNameFromCR(bckt)

	return &cloud.BucketAttrs{
		Name:     bucketName,
		Location: location,
		Properties: &vedrov1alpha1.BucketProperties{
			StorageClass:           spec.StorageClass,
			Labels:                 maps.Clone(spec.Labels),
			Versioning:             spec.Versioning.DeepCopy(),
			PublicAccessPrevention: cloneBool(spec.PublicAccessPrevention),
			Lifecycle:              spec.Lifecycle.DeepCopy(),
		},
	}
}

func NormalizedBucketVersioning(ver *vedrov1alpha1.BucketVersioning) *vedrov1alpha1.BucketVersioning {
	if ver == nil {
		return &vedrov1alpha1.BucketVersioning{
			Enabled: false,
		}
	}
	return ver
}

func NormalizedBucketPAP(pap *bool) *bool {
	if pap == nil {
		return Ptr(false)
	}

	return pap
}

func NormalizedBucketLifecycle(lifecycle *vedrov1alpha1.BucketLifecycle) *vedrov1alpha1.BucketLifecycle {
	if lifecycle == nil || len(lifecycle.Rules) == 0 {
		return &vedrov1alpha1.BucketLifecycle{}
	}
	return lifecycle
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func Ptr[T interface{}](v T) *T {
	return &v
}

func PatchTo[T any](value T) cloud.Change[T] {
	return cloud.Change[T]{Set: true, Value: value}
}
