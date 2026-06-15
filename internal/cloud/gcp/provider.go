package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

type Provider struct {
	bucket *Bucket
}

func New(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedrov1alpha1.ProviderConfig,
) (*Provider, error) {

	client, err := newClient(ctx, kubeClient, cfg)
	if err != nil {
		return nil, err
	}

	p := &Provider{}

	p.bucket = &Bucket{
		client:    &storageClientAdapter{client: client},
		projectId: cfg.Spec.ProjectId,
	}

	return p, nil
}

func newClient(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedrov1alpha1.ProviderConfig,
) (*storage.Client, error) {
	switch cfg.Spec.Method {
	case vedrov1alpha1.AuthMethodWorkloadIdentity:
		return storage.NewClient(ctx)

		//	case vedrov1alpha1.ProviderAuthMethodSecret:
		//		secretRef := cfg.Spec.CredentialsSecretRef
		//		if secretRef == nil {
		//			return nil, fmt.Errorf("spec.credentialsSecretRef is required when auth.method is Secret")
		//		}
		//
		//		var secret corev1.Secret
		//		err := kubeClient.Get(ctx, client.ObjectKey{
		//			Name:      secretRef.Name,
		//			Namespace: secretRef.Namespace,
		//		}, &secret)
		//		if err != nil {
		//			return nil, fmt.Errorf("get credentials secret %s/%s: %w",
		//				secretRef.Namespace,
		//				secretRef.Name,
		//				err,
		//			)
		//		}
		//
		//		key := "key"
		//
		//		credentialsJSON, ok := secret.Data[key]
		//		if !ok {
		//			return nil, fmt.Errorf("credentials secret %s/%s does not contain key %q",
		//				secretRef.Namespace,
		//				secretRef.Name,
		//				key,
		//			)
		//		}
		//
		//		return storage.NewClient(ctx, option.WithCredentialsJSON(credentialsJSON))
		//
	default:
		return nil, fmt.Errorf("unsupported provider auth method %q", cfg.Spec.Method)
	}
}

func (p *Provider) Capabilities() cloud.Capabilities {
	return cloud.Capabilities{
		Bucket: cloud.BucketCapabilities{
			Versioning:                   true,
			LifecycleExpiration:          true,
			PublicAccessPrevention:       true,
			StorageClassArchive:          true,
			StorageClassInfrequentAccess: true,
			Labels:                       true,
		},
	}
}

func (p *Provider) Bucket() cloud.BucketProvider {
	return p.bucket
}
