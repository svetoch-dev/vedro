package cloudtest

import (
	"context"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewPrincipalCR(
	name string,
	mods ...func(*vedro.CloudPrincipal),
) vedro.CloudPrincipal {
	p := vedro.CloudPrincipal{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: vedro.CloudPrincipalSpec{
			ProviderRef: vedro.ProviderConfigReference{Name: "some-provider"},
		},
	}
	for _, m := range mods {
		m(&p)
	}
	return p
}

type FakePrincipalAPI struct {
	GetAttrs    *cloud.PrincipalAttrs
	CreateAttrs *cloud.PrincipalAttrs
	GetErr      error
	CreateErr   error
	DeleteErr   error
}

func (f *FakePrincipalAPI) GetPrincipal(ctx context.Context, _ string) (*cloud.PrincipalAttrs, error) {
	if f.GetErr != nil {
		return nil, f.GetErr
	}
	return f.GetAttrs, nil
}

func (f *FakePrincipalAPI) Close(ctx context.Context) error {
	return nil
}

func (f *FakePrincipalAPI) CreatePrincipal(
	ctx context.Context,
	name string,
) (*cloud.PrincipalAttrs, error) {
	if f.CreateErr != nil {
		return nil, f.CreateErr
	}
	return f.CreateAttrs, nil
}

func (f *FakePrincipalAPI) DeletePrincipal(ctx context.Context, _ string) error {
	return f.DeleteErr
}
