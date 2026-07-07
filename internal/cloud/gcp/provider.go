package gcp

import (
	"context"
	"fmt"
	"regexp"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/validation"
)

const (
	gcpCredentialsSecretKey = "key"
)

var gcpProjectIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

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
		api: &gcsAPI{
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

func (p *Provider) Cleanup(ctx context.Context) error {
	if err := p.bucket.api.Close(ctx); err != nil {
		return err
	}
	return nil
}

func (p *Provider) ValidateProviderConfigSpec(cfg vedro.ProviderConfig) validation.ValidationResult {
	if !gcpProjectIDPattern.MatchString(cfg.Spec.ProjectId) {
		return validation.Invalid("spec.projectId must be 6-30 characters, start with a lowercase letter, contain only lowercase letters, numbers, and dashes, and end with a letter or number")
	}

	v := validation.ValidateLocation(cfg.Spec.Region, nil)

	if !v.Valid {
		return v
	}

	return validation.Valid()
}
