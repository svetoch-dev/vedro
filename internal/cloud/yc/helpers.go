package yc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awssdkconfig "github.com/aws/aws-sdk-go-v2/config"
	awssdkcreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awscompatibility "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1/awscompatibility"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	accessKeyCreateMethod = protoreflect.FullName(
		"yandex.cloud.iam.v1.awscompatibility.AccessKeyService.Create",
	)
	ycWhoAmIUrl = "https://auth.yandex.cloud/oauth/userinfo"
)

func whoAmI(
	ctx context.Context,
	token string,
) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ycWhoAmIUrl, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}
	var r struct {
		SaId string `json:"sub"`
	}

	err = json.NewDecoder(resp.Body).Decode(&r)

	if err != nil {
		return "", err
	}

	return r.SaId, nil
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
