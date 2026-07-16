package yc

import (
	"context"

	"fmt"

	"github.com/svetoch-dev/vedro/internal/cloud"

	iamapi "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	iamsdk "github.com/yandex-cloud/go-sdk/v2/services/iam/v1"
)

type ycPrincipalAPI struct {
	sdk      *ycsdk.SDK
	folderId string
}

func (y *ycPrincipalAPI) findServiceAccount(ctx context.Context, name string) (*iamapi.ServiceAccount, error) {
	client := iamsdk.NewServiceAccountClient(y.sdk)
	response, err := client.List(ctx, &iamapi.ListServiceAccountsRequest{
		FolderId: y.folderId,
		Filter:   fmt.Sprintf(`name = "%s"`, name),
		PageSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("list service accounts: %w", err)
	}

	if len(response.ServiceAccounts) == 0 {
		return nil, cloud.ErrPrincipalNotFound
	}

	return response.ServiceAccounts[0], nil
}

func (y *ycPrincipalAPI) GetPrincipal(ctx context.Context, name string) (*cloud.PrincipalAttrs, error) {
	sa, err := y.findServiceAccount(ctx, name)

	if err != nil {
		return nil, err
	}

	return &cloud.PrincipalAttrs{
		Name: sa.Name,
		Id:   sa.Id,
	}, nil
}

func (y *ycPrincipalAPI) CreatePrincipal(ctx context.Context, name string) (*cloud.PrincipalAttrs, error) {
	client := iamsdk.NewServiceAccountClient(y.sdk)

	op, err := client.Create(ctx, &iamapi.CreateServiceAccountRequest{
		FolderId: y.folderId,
		Name:     name,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"start creating service account %q: %w",
			name,
			err,
		)
	}

	serviceAccount, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create service account %q: %w",
			name,
			err,
		)
	}

	return &cloud.PrincipalAttrs{
		Name: serviceAccount.Name,
		Id:   serviceAccount.Id,
	}, nil

}

func (y *ycPrincipalAPI) DeletePrincipal(ctx context.Context, name string) error {

	sa, err := y.findServiceAccount(ctx, name)
	if err != nil {
		return err
	}

	client := iamsdk.NewServiceAccountClient(y.sdk)

	op, err := client.Delete(ctx, &iamapi.DeleteServiceAccountRequest{
		ServiceAccountId: sa.Id,
	})
	if err != nil {
		return fmt.Errorf(
			"start deleting service account %q: %w",
			name,
			err,
		)
	}

	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf(
			"delete service account %q: %w",
			name,
			err,
		)
	}

	return nil
}

func (y *ycPrincipalAPI) Close(ctx context.Context) error {
	return nil
}
