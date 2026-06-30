package helpers

import (
	"context"
	"fmt"
	"maps"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func BucketNameFromCR(bckt vedro.Bucket) string {
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
	bckt vedro.Bucket,
	caps cloud.BucketCapabilities,
) *cloud.BucketAttrs {
	spec := bckt.Spec
	bucketName := BucketNameFromCR(bckt)

	return &cloud.BucketAttrs{
		Name:     bucketName,
		Location: location,
		Properties: &vedro.BucketProperties{
			StorageClass:           spec.StorageClass,
			Labels:                 maps.Clone(spec.Labels),
			Versioning:             NormalizedBucketVersioning(spec.Versioning.DeepCopy()),
			PublicAccessPrevention: NormalizedBucketPAP(cloneBool(spec.PublicAccessPrevention)),
			Lifecycle:              NormalizedBucketLifecycle(spec.Lifecycle.DeepCopy(), caps),
		},
	}
}

func NormalizedBucketVersioning(ver *vedro.BucketVersioning) *vedro.BucketVersioning {
	if ver == nil {
		return &vedro.BucketVersioning{
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

func NormalizedBucketLifecycle(
	lifecycle *vedro.BucketLifecycle,
	caps cloud.BucketCapabilities,
) *vedro.BucketLifecycle {
	normalized := &vedro.BucketLifecycle{}
	if lifecycle == nil || len(lifecycle.Rules) == 0 {
		return normalized
	}

	for _, rule := range lifecycle.Rules {
		if !rule.Enabled {
			continue
		}
		if !caps.Lifecycle.RuleNames {
			normalized.Rules = append(normalized.Rules,
				vedro.BucketLifecycleRule{
					AgeDays: rule.AgeDays,
					Action:  rule.Action,
					Enabled: rule.Enabled,
				},
			)
		} else {
			normalized.Rules = append(normalized.Rules, rule)
		}
	}
	return normalized
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
