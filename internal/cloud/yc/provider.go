package yc

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"github.com/yandex-cloud/go-sdk/v2/credentials"
	"github.com/yandex-cloud/go-sdk/v2/pkg/iamkey"
	"github.com/yandex-cloud/go-sdk/v2/pkg/options"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/validation"
)

const (
	ycCredentialsSecretKey = "key"
)

var ycProjectIDPattern = regexp.MustCompile(`^b1g[a-z0-9]{17}$`)

type sdkShutdowner interface {
	Shutdown(ctx context.Context) error
}

type Provider struct {
	bucket    *Bucket
	principal *Principal
	sdk       sdkShutdowner
}

func New(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedro.ProviderConfig,
) (*Provider, error) {

	sdk, saID, err := newClient(ctx, kubeClient, cfg)
	if err != nil {
		return nil, err
	}

	ycsApi := &ycsAPI{
		sdk:      sdk,
		folderId: cfg.Spec.ProjectId,
		saId:     saID,
		location: cfg.Spec.Region,
	}
	ycPrincipalApi := &ycPrincipalAPI{
		sdk:      sdk,
		folderId: cfg.Spec.ProjectId,
	}

	p := &Provider{
		bucket: &Bucket{
			api: ycsApi,
		},
		principal: &Principal{
			api: ycPrincipalApi,
		},
		sdk: sdk,
	}

	return p, nil
}

func newClient(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedro.ProviderConfig,
) (*ycsdk.SDK, string, error) {
	switch cfg.Spec.Method {
	case vedro.AuthMethodWorkloadIdentity:

		sdk, err := ycsdk.Build(ctx, options.WithCredentials(credentials.InstanceServiceAccount()))
		if err != nil {
			return nil, "", err
		}
		token, err := sdk.CreateIAMToken(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("WorkloadIdentity: failed to create token: %w", err)
		}

		saId, err := whoAmI(ctx, token.GetIamToken())
		if err != nil {
			return nil, "", fmt.Errorf("WorkloadIdentity: get yc service account id: %w", err)
		}

		return sdk, saId, nil
	case vedro.AuthMethodStaticCredentials:
		secretRef := cfg.Spec.CredentialsSecretRef
		if secretRef == nil {
			return nil, "", fmt.Errorf("spec.credentialsSecretRef is required when auth.method is Secret")
		}

		data, err := helpers.GetSecretData(ctx, kubeClient, *secretRef, ycCredentialsSecretKey)

		if err != nil {
			return nil, "", err
		}

		key, err := iamkey.ReadFromJSONBytes(data[ycCredentialsSecretKey])
		if err != nil {
			return nil, "", fmt.Errorf("StaticCreds: parse yc service account key json: %w", err)
		}

		creds, err := credentials.ServiceAccountKey(key)
		if err != nil {
			return nil, "", fmt.Errorf("StaticCred: create yc service account credentials: %w", err)
		}

		sdk, err := ycsdk.Build(ctx,
			options.WithCredentials(creds),
		)
		return sdk, key.GetServiceAccountId(), err

	default:
		return nil, "", fmt.Errorf("unsupported provider auth method %q", cfg.Spec.Method)
	}
}

func (p *Provider) Capabilities() cloud.Capabilities {
	return cloud.Capabilities{
		Bucket: cloud.BucketCapabilities{
			Versioning: true,
			Lifecycle: cloud.LifecycleCapabilities{
				RuleExpiration: true,
				RuleNames:      true,
			},
			StorageClass: cloud.StorageClassCapabilities{
				Ice:  true,
				Cold: true,
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
	bucketCloseErr := p.bucket.api.Close(ctx)
	sdkShutdownErr := p.sdk.Shutdown(ctx)
	return errors.Join(bucketCloseErr, sdkShutdownErr)

}

func (p *Provider) ValidateProviderConfigSpec(cfg vedro.ProviderConfig) validation.ValidationResult {
	if !ycProjectIDPattern.MatchString(cfg.Spec.ProjectId) {
		return validation.Invalid("spec.projectId must be 5-63 characters, start with a lowercase letter, and contain only lowercase letters, numbers, and dashes")
	}

	v := validation.ValidateLocation(cfg.Spec.Region, nil)

	if !v.Valid {
		return v
	}

	return validation.Valid()
}
