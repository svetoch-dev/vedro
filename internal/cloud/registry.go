package cloud

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud/gcp"
)

func NewProvider(
	ctx context.Context,
	cfg *vedrov1alpha1.ProviderConfig,
	kubeClient client.Client,
) (Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("provider config is nil")
	}

	switch cfg.Spec.Type {
	case vedrov1alpha1.ProviderTypeGCP:
		return gcp.New(ctx, kubeClient, cfg)
	default:
		return nil, fmt.Errorf("unsupported provider type %q", cfg.Spec.Type)
	}
}
