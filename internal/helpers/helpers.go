package helpers

import (
	"context"
	"fmt"

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

func PrincipalNameFromCR(prncpl vedro.CloudPrincipal) string {
	cloudPrincipalName := prncpl.Name

	if prncpl.Spec.Name != "" {
		cloudPrincipalName = prncpl.Spec.Name
	}

	return cloudPrincipalName
}

func BucketNameForDelete(bckt vedro.Bucket) string {
	// check needed if deletion starts before the first
	// successful reconcile
	if bckt.Status.ExternalName == "" {
		return BucketNameFromCR(bckt)
	}

	return bckt.Status.ExternalName
}

func PrincipalNameForDelete(prncpl vedro.CloudPrincipal) string {
	// check needed if deletion starts before the first
	// successful reconcile
	if prncpl.Status.ExternalName == "" {
		return PrincipalNameFromCR(prncpl)
	}
	return prncpl.Status.ExternalName
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

func CloneBool(value *bool) *bool {
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
