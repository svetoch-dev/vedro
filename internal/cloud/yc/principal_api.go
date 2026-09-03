package yc

import (
	"context"
	"errors"

	"fmt"

	"github.com/svetoch-dev/vedro/internal/cloud"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	iamapi "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	iamsdk "github.com/yandex-cloud/go-sdk/v2/services/iam/v1"
)

type ycPrincipalAPI struct {
	sdk      *ycsdk.SDK
	folderId string
}

func (y *ycPrincipalAPI) ValidForManagement(principal cloud.PrincipalSetup) bool {
	return principal.Policy == vedro.PrincipalManagementPolicyManaged &&
		principal.Kind == vedro.PrincipalKindServiceAccount
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

func (y *ycPrincipalAPI) GetPrincipal(ctx context.Context, principal cloud.PrincipalSetup) (*cloud.PrincipalAttrs, error) {
	if principal.Policy == vedro.PrincipalManagementPolicyReference {
		id := ""
		switch principal.Kind {
		case vedro.PrincipalKindServiceAccount:
			id = fmt.Sprintf("serviceAccount:%s", principal.Name)
		case vedro.PrincipalKindUser:
			id = fmt.Sprintf("userAccount:%s", principal.Name)

		}

		return &cloud.PrincipalAttrs{
			Name:   principal.Name,
			Kind:   principal.Kind,
			Policy: principal.Policy,
			Id:     id,
		}, nil
	}

	sa, err := y.findServiceAccount(ctx, principal.Name)

	if err != nil {
		return nil, err
	}

	return &cloud.PrincipalAttrs{
		Name:   sa.Name,
		Id:     fmt.Sprintf("serviceAccount:%s", sa.Id),
		Kind:   principal.Kind,
		Policy: principal.Policy,
	}, nil
}

func (y *ycPrincipalAPI) CreatePrincipal(ctx context.Context, principal cloud.PrincipalSetup) (*cloud.PrincipalAttrs, error) {
	if !y.ValidForManagement(principal) {
		return nil, fmt.Errorf("Principal can only be a managed ServiceAccount")
	}
	client := iamsdk.NewServiceAccountClient(y.sdk)

	op, err := client.Create(ctx, &iamapi.CreateServiceAccountRequest{
		FolderId: y.folderId,
		Name:     principal.Name,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"start creating service account %q: %w",
			principal.Name,
			err,
		)
	}

	serviceAccount, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create service account %q: %w",
			principal.Name,
			err,
		)
	}

	return &cloud.PrincipalAttrs{
		Name:   serviceAccount.Name,
		Id:     fmt.Sprintf("serviceAccount:%s", serviceAccount.Id),
		Kind:   principal.Kind,
		Policy: principal.Policy,
	}, nil

}

func (y *ycPrincipalAPI) DeletePrincipal(ctx context.Context, principal cloud.PrincipalSetup) error {

	if !y.ValidForManagement(principal) {
		return fmt.Errorf("Principal can only be a managed ServiceAccount")
	}

	sa, err := y.findServiceAccount(ctx, principal.Name)
	if err != nil {
		if errors.Is(err, cloud.ErrPrincipalNotFound) {
			return nil
		}

		return err
	}

	client := iamsdk.NewServiceAccountClient(y.sdk)

	op, err := client.Delete(ctx, &iamapi.DeleteServiceAccountRequest{
		ServiceAccountId: sa.Id,
	})
	if err != nil {
		return fmt.Errorf(
			"start deleting service account %q: %w",
			principal.Name,
			err,
		)
	}

	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf(
			"delete service account %q: %w",
			principal.Name,
			err,
		)
	}

	return nil
}

func (y *ycPrincipalAPI) Close(ctx context.Context) error {
	return nil
}
