package gcp

import (
	"context"
	"fmt"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gcpPrincipalAPI struct {
	client    *admin.IamClient
	projectID string
}

func saEmailAndFullName(name, projectId string) (string, string) {
	email := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", name, projectId)
	fullName := fmt.Sprintf(
		"projects/%s/serviceAccounts/%s",
		projectId,
		email,
	)

	return email, fullName
}

func (p *gcpPrincipalAPI) GetPrincipal(ctx context.Context, name string) (*cloud.PrincipalAttrs, error) {
	email, fullName := saEmailAndFullName(name, p.projectID)

	account, err := p.client.GetServiceAccount(ctx, &adminpb.GetServiceAccountRequest{
		Name: fullName,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, cloud.ErrPrincipalNotFound

		}
		return nil, fmt.Errorf("get service account %q: %w", email, err)
	}

	return &cloud.PrincipalAttrs{
		Name: name,
		Id:   account.Email,
	}, nil
}

func (p *gcpPrincipalAPI) CreatePrincipal(ctx context.Context, name string) (*cloud.PrincipalAttrs, error) {
	account, err := p.client.CreateServiceAccount(
		ctx,
		&adminpb.CreateServiceAccountRequest{
			Name:      fmt.Sprintf("projects/%s", p.projectID),
			AccountId: name,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create service account %q in project %q: %w",
			name,
			p.projectID,
			err,
		)
	}

	return &cloud.PrincipalAttrs{
		Name: name,
		Id:   account.Email,
	}, nil
}

func (p *gcpPrincipalAPI) DeletePrincipal(ctx context.Context, name string) error {
	email, fullName := saEmailAndFullName(name, p.projectID)
	err := p.client.DeleteServiceAccount(
		ctx,
		&adminpb.DeleteServiceAccountRequest{
			Name: fullName,
		},
	)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}

		return fmt.Errorf(
			"delete service account %q: %w",
			email,
			err,
		)
	}

	return nil
}

func (p *gcpPrincipalAPI) Close(ctx context.Context) error {
	if p.client == nil {
		return nil
	}
	return p.client.Close()
}
