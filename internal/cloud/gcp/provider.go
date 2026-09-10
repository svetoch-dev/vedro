package gcp

import (
	"context"
	"errors"
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
	iam "google.golang.org/api/iam/v1"
)

const (
	gcpCredentialsSecretKey = "key"
)

var gcpProjectIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

type gcpClients struct {
	storage    *storage.Client
	iamAdmin   *admin.IamClient
	iamService *iam.Service
}

func (o *gcpClients) Close() error {
	var storageCloseErr error
	var iamAdminCloseErr error
	if o.storage != nil {
		storageCloseErr = o.storage.Close()
	}

	if o.iamAdmin != nil {
		iamAdminCloseErr = o.iamAdmin.Close()
	}

	return errors.Join(storageCloseErr, iamAdminCloseErr)
}

type Provider struct {
	bucket        *Bucket
	principal     *Principal
	principalAuth *PrincipalAuth
	bucketAccess  *BucketAccess
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

	gcsApi := &gcsAPI{
		projectID: cfg.Spec.ProjectId,
		client:    clients.storage,
	}

	p.bucket = &Bucket{
		api: gcsApi,
	}

	p.bucketAccess = &BucketAccess{
		api: gcsApi,
	}

	p.principal = &Principal{
		api: &gcpPrincipalAPI{
			projectID: cfg.Spec.ProjectId,
			clients:   clients,
		},
	}

	p.principalAuth = &PrincipalAuth{
		api: &gcpPrincipalAPI{
			projectID: cfg.Spec.ProjectId,
			clients:   clients,
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
		iamAdminClient, err := admin.NewIamClient(ctx)
		if err != nil {
			return nil, errors.Join(
				storageClient.Close(),
				fmt.Errorf("WorkloadIdentity: error getting iam admin client %w", err),
			)
		}

		iamServiceClient, err := iam.NewService(ctx)
		if err != nil {
			return nil, errors.Join(
				storageClient.Close(),
				iamAdminClient.Close(),
				fmt.Errorf("WorkloadIdentity: error getting iam service client %w", err),
			)
		}

		return &gcpClients{
			storage:    storageClient,
			iamAdmin:   iamAdminClient,
			iamService: iamServiceClient,
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

		iamAdminClient, err := admin.NewIamClient(ctx, credentials)
		if err != nil {
			return nil, errors.Join(
				storageClient.Close(),
				fmt.Errorf("StaticCredentials: error getting iam admin client %w", err),
			)
		}

		iamServiceClient, err := iam.NewService(ctx, credentials)
		if err != nil {
			return nil, errors.Join(
				storageClient.Close(),
				iamAdminClient.Close(),
				fmt.Errorf("StaticCredentials: error getting iam service client %w", err),
			)
		}
		return &gcpClients{
			storage:    storageClient,
			iamAdmin:   iamAdminClient,
			iamService: iamServiceClient,
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
		BucketAccess: cloud.BucketAccessCapabilities{
			ObjectReader: true,
			ObjectWriter: true,
			ObjectAdmin:  true,
			BucketAdmin:  true,
		},
		Principal: cloud.PrincipalCapabilities{
			ManagedKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
			ReferencedKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
				vedro.PrincipalKindGroup:          true,
				vedro.PrincipalKindUser:           true,
				vedro.PrincipalKindAllUsers:       true,
			},
		},
		PrincipalAuth: cloud.PrincipalAuthCapabilities{
			StaticCredentials: true,
			WorkloadIdentity:  true,
			WorkloadIdentityKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
			StaticCredentialsKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
		},
	}
}

func (p *Provider) Bucket() cloud.BucketProvider {
	return p.bucket
}

func (p *Provider) Principal() cloud.PrincipalProvider {
	return p.principal
}

func (p *Provider) PrincipalAuth() cloud.PrincipalAuthProvider {
	return p.principalAuth
}

func (p *Provider) Access() cloud.BucketAccessProvider {
	return p.bucketAccess
}

func (p *Provider) Cleanup(ctx context.Context) error {
	bucketCloseErr := p.bucket.api.Close(ctx)
	principalCloseErr := p.principal.api.Close(ctx)
	principalAuthCloseErr := p.principalAuth.api.Close(ctx)
	return errors.Join(bucketCloseErr, principalCloseErr, principalAuthCloseErr)
}

func (p *Provider) ValidateProviderConfigSpec(cfg vedro.ProviderConfig) validation.ValidationResult {
	if !gcpProjectIDPattern.MatchString(cfg.Spec.ProjectId) {
		return validation.Invalid("spec.projectId must be 6-30 characters, start with a lowercase letter, contain only lowercase letters, numbers, and dashes, and end with a letter or number")
	}

	v := validation.ValidateLocation(cfg.Spec.Region, nil)

	if !v.Valid {
		return v
	}

	v = validation.ValidateUsagePolicy(cfg.Spec.UsagePolicy)

	if !v.Valid {
		return v
	}

	return validation.Valid()
}
