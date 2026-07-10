package gcp

import (
	"context"
	"fmt"
	"regexp"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/client"

	admin "cloud.google.com/go/iam/admin/apiv1"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/validation"
)

const (
	gcpCredentialsSecretKey = "key"
)

var gcpProjectIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

type gcpClients struct {
	storage *storage.Client
	iam     *admin.IamClient
}

type Provider struct {
	bucket    *Bucket
	principal *Principal
}

func New(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedro.ProviderConfig,
) (*Provider, error) {

	clients, err := newClient(ctx, kubeClient, cfg)
	if err != nil {
		return nil, err
	}

	p := &Provider{}

	p.bucket = &Bucket{
		api: &gcsAPI{
			projectID: cfg.Spec.ProjectId,
			client:    clients.storage,
		},
	}
	p.principal = &Principal{
		api: &gcpPrincipalAPI{
			client: clients.iam,
		},
	}

	return p, nil
}

func newClient(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedro.ProviderConfig,
) (*gcpClients, error) {
	switch cfg.Spec.Method {
	case vedro.AuthMethodWorkloadIdentity:
		storageClient, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("WorkloadIdentity: error getting storage client %w", err)
		}
		iamClient, err := admin.NewIamClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("WorkloadIdentity: error getting iam client %w", err)
		}

		return &gcpClients{
			storage: storageClient,
			iam:     iamClient,
		}, nil

	case vedro.AuthMethodStaticCredentials:
		secretRef := cfg.Spec.CredentialsSecretRef
		if secretRef == nil {
			return nil, fmt.Errorf("spec.credentialsSecretRef is required when auth.method is Secret")
		}

		data, err := helpers.GetSecretData(ctx, kubeClient, *secretRef, gcpCredentialsSecretKey)

		if err != nil {
			return nil, err
		}

		credentials := option.WithAuthCredentialsJSON(option.ServiceAccount, data[gcpCredentialsSecretKey])
		storageClient, err := storage.NewClient(ctx, credentials)
		if err != nil {
			return nil, fmt.Errorf("StaticCredentials: error getting storage client %w", err)
		}
		iamClient, err := admin.NewIamClient(ctx, credentials)
		if err != nil {
			return nil, fmt.Errorf("StaticCredentials: error getting iam client %w", err)
		}

		return &gcpClients{
			storage: storageClient,
			iam:     iamClient,
		}, nil

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

func (p *Provider) Principal() cloud.PrincipalProvider {
	return p.principal
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
