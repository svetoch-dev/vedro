/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/validation"
)

var _ = Describe("prepareProvider", func() {
	It("reports a missing ProviderConfig without calling the factory", func() {
		factoryCalled := false
		setup, issue := prepareProvider(
			ctx,
			vedro.ProviderConfigReference{Name: "missing-provider"},
			k8sClient,
			func(context.Context, vedro.ProviderConfig, client.Client) (cloud.Provider, error) {
				factoryCalled = true
				return nil, nil
			},
		)

		Expect(issue).NotTo(BeNil())
		Expect(issue.Kind).To(Equal(ProviderResolveFailed))
		Expect(apierrors.IsNotFound(issue.Error)).To(BeTrue())
		Expect(setup.Provider).To(BeNil())
		Expect(setup.Config.Condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(setup.Config.Condition.Reason).To(Equal(conditions.ReasonProviderConfigNotFound))
		Expect(factoryCalled).To(BeFalse())
	})

	It("reports provider factory errors", func() {
		createProviderConfig(ctx)
		factoryErr := errors.New("provider setup failed")

		setup, issue := prepareProvider(
			ctx,
			vedro.ProviderConfigReference{Name: "test-provider"},
			k8sClient,
			func(context.Context, vedro.ProviderConfig, client.Client) (cloud.Provider, error) {
				return nil, factoryErr
			},
		)

		Expect(issue).NotTo(BeNil())
		Expect(issue.Kind).To(Equal(ProviderSettingFailed))
		Expect(issue.Error).To(MatchError(factoryErr))
		Expect(setup.Provider).To(BeNil())
		Expect(setup.Config.Condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(setup.Config.Condition.Reason).To(Equal(conditions.ReasonProviderConfigError))
		Expect(setup.Config.Condition.Message).To(Equal(factoryErr.Error()))
	})

	It("returns the provider and reports an invalid ProviderConfig", func() {
		createProviderConfig(ctx)
		provider := &providerConfigValidationProvider{
			fakeProvider: &fakeProvider{},
			result:       validation.Invalid("invalid provider config"),
		}

		setup, issue := prepareProvider(
			ctx,
			vedro.ProviderConfigReference{Name: "test-provider"},
			k8sClient,
			func(context.Context, vedro.ProviderConfig, client.Client) (cloud.Provider, error) {
				return provider, nil
			},
		)

		Expect(issue).NotTo(BeNil())
		Expect(issue.Kind).To(Equal(ProviderConfigInvalid))
		Expect(issue.Error).To(HaveOccurred())
		Expect(setup.Provider).To(BeIdenticalTo(provider))
		Expect(setup.Config.Condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(setup.Config.Condition.Reason).To(Equal(conditions.ReasonProviderConfigInvalidSpec))
		Expect(setup.Config.Condition.Message).To(Equal("invalid provider config"))
	})

	It("returns a configured provider with a ready condition", func() {
		createProviderConfig(ctx)
		provider := &providerConfigValidationProvider{
			fakeProvider: &fakeProvider{},
			result:       validation.Valid(),
		}

		setup, issue := prepareProvider(
			ctx,
			vedro.ProviderConfigReference{Name: "test-provider"},
			k8sClient,
			func(context.Context, vedro.ProviderConfig, client.Client) (cloud.Provider, error) {
				return provider, nil
			},
		)

		Expect(issue).To(BeNil())
		Expect(setup.Provider).To(BeIdenticalTo(provider))
		Expect(setup.Config.Name).To(Equal("test-provider"))
		Expect(setup.Config.Condition.Status).To(Equal(metav1.ConditionTrue))
		Expect(setup.Config.Condition.Reason).To(Equal(conditions.ReasonProviderConfigReconciled))
		Expect(setup.Config.Condition.Message).To(Equal("ProviderConfig Reconciled"))
	})
})

type providerConfigValidationProvider struct {
	*fakeProvider
	result validation.ValidationResult
}

func (p *providerConfigValidationProvider) ValidateProviderConfigSpec(
	vedro.ProviderConfig,
) validation.ValidationResult {
	return p.result
}
