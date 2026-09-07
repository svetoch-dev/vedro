package helpers

import (
	"context"
	"fmt"
	"strings"

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

// Parse string in format <principal_type>:<principal_name>.
// Returns <principal_type>, <principal_name>
// Examples:
//  1. "user:some_user@example.com"
//     returns "user", "some_user@example.com"
//  2. "serviceAccount:a121dawd1faagty"
//     returns "serviceAccount", "a121dawd1faagty"
//  3. "a121dawd1faagty"
//     returns "", "a121dawd1faagty"
func ParseIAMMemberString(input string) (string, string) {
	parts := strings.SplitN(input, ":", 2)
	if len(parts) != 2 {
		return "", parts[0]
	}

	return parts[0], parts[1]
}

func PrincipalNameFromCR(prncpl vedro.CloudPrincipal) string {
	cloudPrincipalName := ""

	if prncpl.Spec.ManagementPolicy == vedro.PrincipalManagementPolicyManaged &&
		prncpl.Spec.Managed != nil &&
		prncpl.Spec.Managed.Name != "" {
		cloudPrincipalName = prncpl.Spec.Managed.Name
	}

	if prncpl.Spec.ManagementPolicy == vedro.PrincipalManagementPolicyReference &&
		prncpl.Spec.Reference != nil &&
		prncpl.Spec.Reference.Name != "" {
		cloudPrincipalName = prncpl.Spec.Reference.Name
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
