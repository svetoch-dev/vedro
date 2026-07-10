package gcp

import (
	"context"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

type gcpPrincipalAPI struct {
	client *admin.IamClient
}

var _ cloud.PrincipalAPI = (*gcpPrincipalAPI)(nil)

func (p *gcpPrincipalAPI) GetPrincipal(ctx context.Context, name string) (*cloud.PrincipalAttrs, error) {
	return nil, nil
}

func (p *gcpPrincipalAPI) CreatePrincipal(ctx context.Context, name string, attrs cloud.PrincipalAttrs) error {
	return nil
}

func (p *gcpPrincipalAPI) DeletePrincipal(ctx context.Context, name string) error {
	return nil
}
