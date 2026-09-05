package yc

import (
	"context"
	"errors"
	"strings"

	"fmt"

	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	iamapi "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	organizationmanagerapi "github.com/yandex-cloud/go-genproto/yandex/cloud/organizationmanager/v1"
	resourcemanagerapi "github.com/yandex-cloud/go-genproto/yandex/cloud/resourcemanager/v1"
	organizationmanagersdk "github.com/yandex-cloud/go-sdk/services/organizationmanager/v1"
	resourcemanagersdk "github.com/yandex-cloud/go-sdk/services/resourcemanager/v1"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	iamsdk "github.com/yandex-cloud/go-sdk/v2/services/iam/v1"
)

type ycPrincipalAPI struct {
	sdk      *ycsdk.SDK
	folderId string
}

func (y *ycPrincipalAPI) findServiceAccount(ctx context.Context, folderId string, name string) (*iamapi.ServiceAccount, error) {
	client := iamsdk.NewServiceAccountClient(y.sdk)
	response, err := client.List(ctx, &iamapi.ListServiceAccountsRequest{
		FolderId: folderId,
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

func (y *ycPrincipalAPI) findCloud(ctx context.Context, folderId string) (*resourcemanagerapi.Cloud, error) {
	folderClient := resourcemanagersdk.NewFolderClient(y.sdk)

	folder, err := folderClient.Get(ctx, &resourcemanagerapi.GetFolderRequest{
		FolderId: folderId,
	})
	if err != nil {
		return nil, err
	}

	cloudClient := resourcemanagersdk.NewCloudClient(y.sdk)

	yccloud, err := cloudClient.Get(
		ctx,
		&resourcemanagerapi.GetCloudRequest{
			CloudId: folder.CloudId,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get cloud %q: %w", folder.CloudId, err)
	}

	return yccloud, nil
}

func (y *ycPrincipalAPI) findFolder(
	ctx context.Context,
	cloudID string,
	name string,
) (*resourcemanagerapi.Folder, error) {
	folderClient := resourcemanagersdk.NewFolderClient(y.sdk)

	resp, err := folderClient.List(
		ctx,
		&resourcemanagerapi.ListFoldersRequest{
			CloudId: cloudID,
			Filter:  fmt.Sprintf(`name="%s"`, name),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}

	if len(resp.Folders) == 0 {
		return nil, fmt.Errorf(
			"folder %q not found in cloud %q",
			name,
			cloudID,
		)
	}

	return resp.Folders[0], nil
}

func (y *ycPrincipalAPI) findUser(
	ctx context.Context,
	orgID string,
	email string,
) (*organizationmanagerapi.ListMembersResponse_OrganizationUser, error) {
	userClient := organizationmanagersdk.NewUserClient(y.sdk)

	var pageToken string

	for {
		resp, err := userClient.ListMembers(
			ctx,
			&organizationmanagerapi.ListMembersRequest{
				OrganizationId: orgID,
				PageSize:       1000,
				PageToken:      pageToken,
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"list users in organization %q: %w",
				orgID,
				err,
			)
		}

		for _, user := range resp.Users {
			if strings.EqualFold(user.SubjectClaims.Email, email) {
				return user, nil
			}
		}

		if resp.NextPageToken == "" {
			break
		}

		pageToken = resp.NextPageToken
	}

	return nil, cloud.ErrPrincipalNotFound
}

func (y *ycPrincipalAPI) GetPrincipal(ctx context.Context, principal cloud.PrincipalSetup) (*cloud.PrincipalAttrs, error) {
	if principal.Policy == vedro.PrincipalManagementPolicyReference {
		id := ""
		switch principal.Kind {
		case vedro.PrincipalKindServiceAccount:
			yccloud, err := y.findCloud(ctx, y.folderId)
			if err != nil {
				return nil, err
			}
			//In this case we pass principal.Name as <folder_name>:<service_account_name>
			//so we can use helpers.ParseIAMMemberString because the string is the same
			principalFolderName, principalName := helpers.ParseIAMMemberString(principal.Name)
			folder, err := y.findFolder(ctx, yccloud.Id, principalFolderName)
			if err != nil {
				return nil, err
			}

			sa, err := y.findServiceAccount(ctx, folder.Id, principalName)
			if err != nil {
				return nil, err
			}

			id = fmt.Sprintf("serviceAccount:%s", sa.Id)
		case vedro.PrincipalKindUser:
			yccloud, err := y.findCloud(ctx, y.folderId)
			if err != nil {
				return nil, err
			}

			if yccloud.OrganizationId == "" {
				return nil, fmt.Errorf(
					"cloud %q does not have an organization ID",
					yccloud.Id,
				)
			}

			user, err := y.findUser(ctx, yccloud.OrganizationId, principal.Name)
			if err != nil {
				return nil, err
			}

			id = fmt.Sprintf("userAccount:%s", user.SubjectClaims.Sub)
		default:
			return nil, fmt.Errorf("unknown principal kind %s", principal.Kind)
		}

		return &cloud.PrincipalAttrs{
			Name:   principal.Name,
			Kind:   principal.Kind,
			Policy: principal.Policy,
			Id:     id,
		}, nil
	}

	sa, err := y.findServiceAccount(ctx, y.folderId, principal.Name)
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
	sa, err := y.findServiceAccount(ctx, y.folderId, principal.Name)
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
