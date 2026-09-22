package helpers

import (
	"context"
	"fmt"
	"strings"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
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

func RemoveAllOwnerRefs(
	ctx context.Context,
	kubeClient client.Client,
	name types.NamespacedName,
	obj client.Object,
) error {
	if err := kubeClient.Get(ctx, name, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf(
			"error getting obj %s.%s: %w", name.Namespace, name.Name, err,
		)
	}

	obj.SetOwnerReferences(nil)

	if err := kubeClient.Update(ctx, obj); err != nil {
		return fmt.Errorf(
			"error removing owner ref for obj %s.%s: %w", name.Namespace, name.Name, err,
		)
	}

	return nil
}

func CreateOrUpdateOwned(
	ctx context.Context,
	kubeClient client.Client,
	obj client.Object,
	owner client.Object,
) error {
	existing := obj.DeepCopyObject().(client.Object)
	key := client.ObjectKeyFromObject(obj)
	err := kubeClient.Get(ctx, key, existing)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return kubeClient.Create(
			ctx,
			obj,
		)
	}

	gvk, err := apiutil.GVKForObject(owner, kubeClient.Scheme())
	if err != nil {
		return err
	}
	yes := false
	for _, ref := range existing.GetOwnerReferences() {
		if ref.APIVersion == gvk.GroupVersion().String() &&
			ref.Kind == gvk.Kind &&
			ref.Name == owner.GetName() &&
			ref.UID == owner.GetUID() {
			yes = true
			break
		}
	}

	if !yes {
		return fmt.Errorf(
			"%s/%s is not owned by %s/%s",
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
			owner.GetObjectKind().GroupVersionKind().Kind,
			owner.GetName(),
		)

	}

	return kubeClient.Patch(
		ctx,
		obj,
		client.Apply, //nolint:staticcheck //We need this beacuase we want i generic obj api
		client.FieldOwner(owner.GetName()),
		client.ForceOwnership,
	)
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
