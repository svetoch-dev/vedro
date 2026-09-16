package gcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	iam "google.golang.org/api/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gcpPrincipalAPI struct {
	clients   *gcpClients
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

func (p *gcpPrincipalAPI) saCreateKey(
	ctx context.Context,
	fullSaName string,
) (*iam.ServiceAccountKey, error) {
	key, err := p.clients.iamService.
		Projects.
		ServiceAccounts.
		Keys.Create(fullSaName, &iam.CreateServiceAccountKeyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("create service account key: %w", err)
	}

	return key, nil
}

func (p *gcpPrincipalAPI) saGetKey(
	ctx context.Context,
	keyId string,
) (*iam.ServiceAccountKey, error) {
	key, err := p.clients.iamService.Projects.ServiceAccounts.Keys.Get(keyId).
		Context(ctx).
		Do()
	if err != nil {
		if isGoogleAPINotFound(err) {
			return nil, cloud.ErrAuthNotFound
		}

		return nil, fmt.Errorf("get service account key: %w", err)
	}

	return key, nil
}

func (p *gcpPrincipalAPI) saDeleteKey(
	ctx context.Context,
	keyId string,
) error {
	if _, err := p.clients.iamService.
		Projects.
		ServiceAccounts.
		Keys.
		Delete(keyId).
		Context(ctx).
		Do(); err != nil {
		if isGoogleAPINotFound(err) {
			return nil
		}
		return fmt.Errorf("delete service account key: %w", err)
	}

	return nil
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
		case vedro.PrincipalKindAllUsers:
			id = "allUsers"
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

	account, err := p.clients.iamAdmin.GetServiceAccount(ctx, &adminpb.GetServiceAccountRequest{
		Name: fullName,
	})
	if err != nil {
		if isGoogleAPINotFound(err) {
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
	account, err := p.clients.iamAdmin.CreateServiceAccount(
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
	err := p.clients.iamAdmin.DeleteServiceAccount(
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

func (p *gcpPrincipalAPI) GetPrincipalAuth(
	ctx context.Context,
	principalAuth cloud.PrincipalAuthSetup,
) (*cloud.PrincipalAuthResult, error) {
	if principalAuth.Method == vedro.AuthMethodStaticCredentials {
		key, err := p.saGetKey(ctx, principalAuth.CredentialsID)
		if err != nil {
			return nil, err
		}
		return &cloud.PrincipalAuthResult{
			Method:        principalAuth.Method,
			CredentialsID: key.Name,
		}, nil
	}
	return nil, fmt.Errorf("Method %s is not supported", principalAuth.Method)
}
func (p *gcpPrincipalAPI) CreatePrincipalAuth(
	ctx context.Context,
	principalAuth cloud.PrincipalAuthSetup,
) (*cloud.PrincipalAuthResult, error) {
	_, email := helpers.ParseIAMMemberString(principalAuth.ServiceAccountID)
	fullName := fmt.Sprintf(
		"projects/%s/serviceAccounts/%s",
		p.projectID,
		email,
	)

	if principalAuth.Method == vedro.AuthMethodStaticCredentials {
		key, err := p.saCreateKey(ctx, fullName)
		if err != nil {
			return nil, err
		}

		data, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
		if err != nil {
			return nil, fmt.Errorf("decode private key: %w", err)
		}

		return &cloud.PrincipalAuthResult{
			Method:        principalAuth.Method,
			CredentialsID: key.Name,
			SecretData: map[string][]byte{
				"credentials.json": data,
			},
		}, nil
	}

	return nil, fmt.Errorf("Method %s is not supported", principalAuth.Method)
}

func (p *gcpPrincipalAPI) DeletePrincipalAuth(
	ctx context.Context,
	principalAuth cloud.PrincipalAuthSetup,
) error {
	if principalAuth.Method == vedro.AuthMethodStaticCredentials {
		err := p.saDeleteKey(ctx, principalAuth.CredentialsID)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("Method %s is not supported", principalAuth.Method)
}

func (p *gcpPrincipalAPI) Close(ctx context.Context) error {
	if p.clients == nil {
		return nil
	}
	return p.clients.Close()
}
