package yc

import (
	"context"
	"fmt"

	awscompatibility "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1/awscompatibility"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"github.com/yandex-cloud/go-sdk/v2/credentials"
	"github.com/yandex-cloud/go-sdk/v2/pkg/iamkey"
	"github.com/yandex-cloud/go-sdk/v2/pkg/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type staticS3AccessKey struct {
	accessKeyID     string
	secretAccessKey string
	id              string
}

type Provider struct {
	bucket    *Bucket
	accessKey *staticS3AccessKey
	sdk       *ycsdk.SDK
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
	p.sdk = sdk

	accessKey, err := createStaticS3AccessKey(ctx, sdk, saID)

	if err != nil {
		return nil, err
	}

	p.accessKey = accessKey

	s3Client, err := newS3Client(ctx, accessKey, cfg.Spec.Region)
	if err != nil {
		cleanupErr := deleteStaticS3AccessKey(ctx, sdk, accessKey.id)
		if cleanupErr != nil {
			return nil, fmt.Errorf("create yc s3 client: %w; cleanup static access key: %v", err, cleanupErr)
		}

		return nil, err
	}

	p.bucket = &Bucket{
		api: &aws.S3API{
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
		logger.V(1).Info("creating yc client")
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
			return nil, "", fmt.Errorf("parse yc service account key json: %w", err)
		}

		creds, err := credentials.ServiceAccountKey(key)
		if err != nil {
			return nil, "", fmt.Errorf("create yc service account credentials: %w", err)
		}

		sdk, err := ycsdk.Build(ctx,
			options.WithCredentials(creds),
		)
		return sdk, key.GetServiceAccountId(), err

	default:
		return nil, "", fmt.Errorf("unsupported provider auth method %q", cfg.Spec.Method)
	}
}

func newS3Client(
	ctx context.Context,
	accessKey *staticS3AccessKey,
	region string,
) (*s3.Client, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("creating yc s3 client")
	cfg, err := awssdkconfig.LoadDefaultConfig(ctx,
		awssdkconfig.WithRegion(region),
		awssdkconfig.WithCredentialsProvider(
			awssdkcreds.NewStaticCredentialsProvider(
				accessKey.accessKeyID,
				accessKey.secretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load yc s3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = awssdk.String("https://storage.yandexcloud.net")
		o.UsePathStyle = true
	})

	return client, nil
}

func createStaticS3AccessKey(
	ctx context.Context,
	sdk *ycsdk.SDK,
	serviceAccountID string,
) (*staticS3AccessKey, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("creating yc static credentials")

	conn, err := sdk.GetConnection(ctx, accessKeyCreateMethod)
	if err != nil {
		return nil, fmt.Errorf("get yc iam awscompatibility connection: %w", err)
	}
	client := awscompatibility.NewAccessKeyServiceClient(conn)

	resp, err := client.Create(ctx, &awscompatibility.CreateAccessKeyRequest{
		ServiceAccountId: serviceAccountID,
	})
	if err != nil {
		return nil, fmt.Errorf("create yc static access key: %w", err)
	}

	return &staticS3AccessKey{
		accessKeyID:     resp.GetAccessKey().GetKeyId(),
		secretAccessKey: resp.GetSecret(),
		id:              resp.GetAccessKey().GetId(),
	}, nil
}

func deleteStaticS3AccessKey(
	ctx context.Context,
	sdk *ycsdk.SDK,
	keyId string,
) error {
	conn, err := sdk.GetConnection(ctx, accessKeyCreateMethod)
	if err != nil {
		return fmt.Errorf("get yc iam awscompatibility connection: %w", err)
	}
	client := awscompatibility.NewAccessKeyServiceClient(conn)
	_, err = client.Delete(ctx, &awscompatibility.DeleteAccessKeyRequest{
		AccessKeyId: keyId,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return fmt.Errorf("delete yc static access key %q: %w", keyId, err)
	}

	return nil

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

func (p *Provider) Cleanup(ctx context.Context) error {
	return deleteStaticS3AccessKey(ctx, p.sdk, p.accessKey.id)
}
