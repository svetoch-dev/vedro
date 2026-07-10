package yc

import (
	"context"

	"github.com/svetoch-dev/vedro/internal/cloud"
)

var _ cloud.PrincipalAPI = (*ycAPI)(nil)

func (y *ycAPI) GetPrincipal(ctx context.Context, name string) (*cloud.PrincipalAttrs, error) {
	return nil, nil
}

func (y *ycAPI) CreatePrincipal(ctx context.Context, name string, attrs cloud.PrincipalAttrs) error {
	return nil
}

func (y *ycAPI) DeletePrincipal(ctx context.Context, name string) error {
	return nil
}
