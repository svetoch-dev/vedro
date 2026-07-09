package registry

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/cloud/gcp"
	"github.com/svetoch-dev/vedro/internal/cloud/yc"
)

func NewProvider(
	ctx context.Context,
	cfg vedro.ProviderConfig,
	kubeClient client.Client,
) (cloud.Provider, error) {
	switch cfg.Spec.Type {
	case vedro.ProviderTypeGCP:
		return gcp.New(ctx, kubeClient, cfg)
	case vedro.ProviderTypeYandexCloud:
		return yc.New(ctx, kubeClient, cfg)
	default:
		return nil, fmt.Errorf("unsupported provider type %q", cfg.Spec.Type)
	}
}
