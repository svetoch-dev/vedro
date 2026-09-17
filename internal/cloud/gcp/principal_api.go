package gcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	workloadIdentityRole = "roles/iam.workloadIdentityUser"
	k8sAnnotationKey     = "iam.gke.io/gcp-service-account"
)

type gcpPrincipalAPI struct {
	clients   *gcpClients
	projectID string
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
		key, err := saGetKey(ctx, p.clients.iamService, principalAuth.CredentialsID)
		if err != nil {
			return nil, err
		}
		return &cloud.PrincipalAuthResult{
			Method:        principalAuth.Method,
			CredentialsID: key.Name,
		}, nil
	}

	if principalAuth.Method == vedro.AuthMethodWorkloadIdentity {
		k8sServiceAccount := principalAuth.K8sServiceAccount
		if k8sServiceAccount == nil {
			return nil, fmt.Errorf("serviceAccountRef is mandatory set with method=WorkloadIdentity")
		}
		_, email := helpers.ParseIAMMemberString(principalAuth.ServiceAccountID)
		yes, err := hasServiceAccountIAMBinding(
			ctx,
			p.clients.iamService,
			email,
			workloadIdentityRole,
			principalAuth.CredentialsID,
		)

		if err != nil {
			return nil, err
		}

		if !yes {
			return nil, cloud.ErrAuthNotFound
		}

		return &cloud.PrincipalAuthResult{
			Method:        principalAuth.Method,
			CredentialsID: principalAuth.CredentialsID,
			ServiceAccountPatch: &corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      k8sServiceAccount.Name,
					Namespace: k8sServiceAccount.Namespace,
					Annotations: map[string]string{
						k8sAnnotationKey: email,
					},
				},
			},
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
		key, err := saCreateKey(ctx, p.clients.iamService, fullName)
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

	if principalAuth.Method == vedro.AuthMethodWorkloadIdentity {
		k8sServiceAccount := principalAuth.K8sServiceAccount
		if k8sServiceAccount == nil {
			return nil, fmt.Errorf("serviceAccountRef is mandatory set with method=WorkloadIdentity")
		}
		principal := fmt.Sprintf(
			"serviceAccount:%s.svc.id.goog[%s/%s]",
			p.projectID,
			k8sServiceAccount.Namespace,
			k8sServiceAccount.Name,
		)
		err := grantServiceAccountIAMBinding(
			ctx,
			p.clients.iamService,
			email,
			workloadIdentityRole,
			principal,
		)
		if err != nil {
			return nil, err
		}

		return &cloud.PrincipalAuthResult{
			Method:        principalAuth.Method,
			CredentialsID: principal,
			ServiceAccountPatch: &corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      k8sServiceAccount.Name,
					Namespace: k8sServiceAccount.Namespace,
					Annotations: map[string]string{
						k8sAnnotationKey: email,
					},
				},
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
		err := saDeleteKey(ctx, p.clients.iamService, principalAuth.CredentialsID)
		if err != nil {
			return err
		}
		return nil
	}

	if principalAuth.Method == vedro.AuthMethodWorkloadIdentity {
		_, email := helpers.ParseIAMMemberString(principalAuth.ServiceAccountID)
		err := revokeServiceAccountIAMBinding(
			ctx,
			p.clients.iamService,
			email,
			workloadIdentityRole,
			principalAuth.CredentialsID,
		)

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
