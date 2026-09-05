package gcp

import (
	"context"
	"fmt"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
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

func (p *gcpPrincipalAPI) GetPrincipal(ctx context.Context, principal cloud.PrincipalSetup) (*cloud.PrincipalAttrs, error) {

	if principal.Policy == vedro.PrincipalManagementPolicyReference {
		id := ""
		switch principal.Kind {
		case vedro.PrincipalKindGroup:
			id = fmt.Sprintf("group:%s", principal.Name)
		case vedro.PrincipalKindServiceAccount:
			id = fmt.Sprintf("serviceAccount:%s", principal.Name)
		case vedro.PrincipalKindUser:
			id = fmt.Sprintf("user:%s", principal.Name)
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

	email, fullName := saEmailAndFullName(principal.Name, p.projectID)

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
		Name:   principal.Name,
		Id:     fmt.Sprintf("serviceAccount:%s", account.Email),
		Kind:   principal.Kind,
		Policy: principal.Policy,
	}, nil
}

func (p *gcpPrincipalAPI) CreatePrincipal(ctx context.Context, principal cloud.PrincipalSetup) (*cloud.PrincipalAttrs, error) {
	account, err := p.client.CreateServiceAccount(
		ctx,
		&adminpb.CreateServiceAccountRequest{
			Name:      fmt.Sprintf("projects/%s", p.projectID),
			AccountId: principal.Name,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create service account %q in project %q: %w",
			principal.Name,
			p.projectID,
			err,
		)
	}

	return &cloud.PrincipalAttrs{
		Name:   principal.Name,
		Id:     fmt.Sprintf("serviceAccount:%s", account.Email),
		Kind:   principal.Kind,
		Policy: principal.Policy,
	}, nil
}

func (p *gcpPrincipalAPI) DeletePrincipal(ctx context.Context, principal cloud.PrincipalSetup) error {
	email, fullName := saEmailAndFullName(principal.Name, p.projectID)
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
