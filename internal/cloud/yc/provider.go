package yc

import (
	"context"
	"encoding/json"
	"fmt"

	awscompatibility "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1/awscompatibility"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"github.com/yandex-cloud/go-sdk/v2/credentials"
	"github.com/yandex-cloud/go-sdk/v2/pkg/options"
	"google.golang.org/protobuf/reflect/protoreflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awssdkconfig "github.com/aws/aws-sdk-go-v2/config"
	awssdkcreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/cloud/aws"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

const (
	ycCredentialsSecretKey = "key"
	accessKeyCreateMethod  = protoreflect.FullName(
		"yandex.cloud.iam.v1.awscompatibility.AccessKeyService.Create",
	)
)

type Provider struct {
	bucket    *Bucket
	accessKey *aws.AccessKey
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

	p := &Provider{}

	accessKey, err := createStaticAccessKey(ctx, sdk, saID)

	if err != nil {
		return nil, err
	}

	p.accessKey = accessKey

	s3Client, err := newYandexS3Client(ctx, accessKey, cfg.Spec.Region)

	p.bucket = &Bucket{
		api: &aws.AwsAPI{
			Client: s3Client,
		},
	}

	return p, nil
}

func newClient(
	ctx context.Context,
	kubeClient client.Client,
	cfg vedro.ProviderConfig,
) (*ycsdk.SDK, string, error) {
	switch cfg.Spec.Method {
	//case vedro.AuthMethodWorkloadIdentity:
	//	return storage.NewClient(ctx)
	case vedro.AuthMethodStaticCredentials:
		logger := log.FromContext(ctx)
		logger.V(1).Info("creating client")
		secretRef := cfg.Spec.CredentialsSecretRef
		if secretRef == nil {
			return nil, "", fmt.Errorf("spec.credentialsSecretRef is required when auth.method is Secret")
		}

		data, err := helpers.GetSecretData(ctx, kubeClient, *secretRef, ycCredentialsSecretKey)

		if err != nil {
			return nil, "", err
		}

		var key struct {
			ServiceAccountID string `json:"service_account_id"`
		}

		if err := json.Unmarshal(data[ycCredentialsSecretKey], &key); err != nil {
			return nil, "", fmt.Errorf("parse service account key: %w", err)
		}

		sdk, err := ycsdk.Build(ctx,
			options.WithCredentials(credentials.IAMToken(string(data[ycCredentialsSecretKey]))),
		)
		return sdk, key.ServiceAccountID, err

	default:
		return nil, "", fmt.Errorf("unsupported provider auth method %q", cfg.Spec.Method)
	}
}

func newYandexS3Client(
	ctx context.Context,
	accessKey *aws.AccessKey,
	region string,
) (*s3.Client, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("creating s3 client")
	cfg, err := awssdkconfig.LoadDefaultConfig(ctx,
		awssdkconfig.WithRegion(region),
		awssdkconfig.WithCredentialsProvider(
			awssdkcreds.NewStaticCredentialsProvider(
				accessKey.AccessKeyID,
				accessKey.SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load yandex s3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = awssdk.String("https://storage.yandexcloud.net")
		o.UsePathStyle = true
	})

	return client, nil
}

func createStaticAccessKey(
	ctx context.Context,
	sdk *ycsdk.SDK,
	serviceAccountID string,
) (*aws.AccessKey, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("creating aws static credentials")

	conn, err := sdk.GetConnection(ctx, accessKeyCreateMethod)
	if err != nil {
		return nil, fmt.Errorf("get yandex iam awscompatibility connection: %w", err)
	}
	client := awscompatibility.NewAccessKeyServiceClient(conn)

	resp, err := client.Create(ctx, &awscompatibility.CreateAccessKeyRequest{
		ServiceAccountId: serviceAccountID,
	})
	if err != nil {
		return nil, fmt.Errorf("create yandex static access key: %w", err)
	}

	return &aws.AccessKey{
		AccessKeyID:     resp.AccessKey.KeyId,
		SecretAccessKey: resp.Secret,
	}, nil
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
