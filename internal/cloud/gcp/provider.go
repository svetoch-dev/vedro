package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

const (
	gcpCredentialsSecretKey = "key"
)

type Provider struct {
	bucket *Bucket
}

func New(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedro.ProviderConfig,
) (*Provider, error) {

	gcsClient, err := newClient(ctx, kubeClient, cfg)
	if err != nil {
		return nil, err
	}

	p := &Provider{}

	p.bucket = &Bucket{
		api: &bucketAPI{
			projectID: cfg.Spec.ProjectId,
			client:    gcsClient,
		},
	}

	return p, nil
}

func newClient(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedro.ProviderConfig,
) (*storage.Client, error) {
	switch cfg.Spec.Method {
	case vedro.AuthMethodWorkloadIdentity:
		return storage.NewClient(ctx)
	case vedro.AuthMethodStaticCredentials:
		secretRef := cfg.Spec.CredentialsSecretRef
		if secretRef == nil {
			return nil, fmt.Errorf("spec.credentialsSecretRef is required when auth.method is Secret")
		}

		data, err := helpers.GetSecretData(ctx, kubeClient, *secretRef, gcpCredentialsSecretKey)

		if err != nil {
			return nil, err
		}

		return storage.NewClient(ctx, option.WithAuthCredentialsJSON(option.ServiceAccount, data[gcpCredentialsSecretKey]))

	default:
		return nil, fmt.Errorf("unsupported provider auth method %q", cfg.Spec.Method)
	}
}

func (p *Provider) Capabilities() cloud.Capabilities {
	return cloud.Capabilities{
		Bucket: cloud.BucketCapabilities{
			Versioning: true,
			Lifecycle: cloud.LifecycleCapabilities{
				RuleExpiration: true,
			},
			PublicAccessPrevention: true,
			StorageClass: cloud.StorageClassCapabilities{
				Ice:  true,
				Cold: true,
				Warm: true,
			},
			Labels: true,
		},
	}
}

func (p *Provider) Bucket() cloud.BucketProvider {
	return p.bucket
}
