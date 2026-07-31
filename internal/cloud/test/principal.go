package cloudtest

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

// PrincipalProviderTests registers the provider-agnostic EnsurePrincipal and
// DeletePrincipal specs. Call it from each provider's Ginkgo suite, e.g.:
//
//	var _ = cloudtest.PrincipalProviderTests(cloudtest.Config{...})
func PrincipalProviderTests(cfg Config) bool {
	newPrincipalCR := func(mods ...func(*vedro.CloudPrincipal)) vedro.CloudPrincipal {
		return NewPrincipalCR("my-principal", mods...)
	}

	Describe("PrincipalProvider.EnsurePrincipal", func() {
		var (
			ctx       context.Context
			fake      *FakePrincipalAPI
			principal cloud.PrincipalProvider
		)
		pattrs := &cloud.PrincipalAttrs{
			Name: "my-principal",
			Id:   "adadadadaw",
		}

		BeforeEach(func() {
			ctx = context.Background()
			fake = &FakePrincipalAPI{}
			principal = cfg.NewPrincipal(fake)
		})

		It("creates a principal when it does not exist", func() {
			fake.GetErr = cloud.ErrPrincipalNotFound
			fake.CreateAttrs = pattrs

			prncpl := newPrincipalCR()

			attrs, err := principal.EnsurePrincipal(ctx, prncpl)
			Expect(err).NotTo(HaveOccurred())
			Expect(attrs).NotTo(BeNil())
			Expect(attrs).To(Equal(fake.CreateAttrs))
		})
		It("Return error and nil if a principal creation fails", func() {
			fake.GetErr = cloud.ErrPrincipalNotFound
			fake.GetErr = errors.New("some error")
			fake.CreateAttrs = pattrs

			prncpl := newPrincipalCR()

			attrs, err := principal.EnsurePrincipal(ctx, prncpl)
			Expect(err).To(HaveOccurred())
			Expect(attrs).To(BeNil())
		})
		It("Return error and nil if get principal fails", func() {
			fake.GetErr = errors.New("some error")
			fake.GetAttrs = pattrs

			prncpl := newPrincipalCR()

			attrs, err := principal.EnsurePrincipal(ctx, prncpl)
			Expect(err).To(HaveOccurred())
			Expect(attrs).To(BeNil())
		})
		It("Return attrs if get principal succeeds", func() {
			fake.GetAttrs = pattrs

			prncpl := newPrincipalCR()

			attrs, err := principal.EnsurePrincipal(ctx, prncpl)
			Expect(err).NotTo(HaveOccurred())
			Expect(attrs).NotTo(BeNil())
			Expect(attrs).To(Equal(fake.GetAttrs))
		})

	})
	Describe("PrincipalProvider.DeletePrincipal", func() {
		var (
			ctx       context.Context
			fake      *FakePrincipalAPI
			principal cloud.PrincipalProvider
		)

		BeforeEach(func() {
			ctx = context.Background()
			fake = &FakePrincipalAPI{}
			principal = cfg.NewPrincipal(fake)
		})

		It("returns an error when bucket deletion fails", func() {
			fake.DeleteErr = errors.New("bucket delete failed")
			prncpl := newPrincipalCR()

			err := principal.DeletePrincipal(ctx, prncpl)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bucket delete failed"))
		})
		It("returns no error when bucket deletion succeeds", func() {
			prncpl := newPrincipalCR()

			err := principal.DeletePrincipal(ctx, prncpl)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	return true
}

func PrincipalValidationTests(cfg Config) bool {
	newPrincipalCR := func(mods ...func(*vedro.CloudPrincipal)) vedro.CloudPrincipal {
		return NewPrincipalCR("my-principal", mods...)
	}
	Describe("Principal.ValidatePrincipalSpec", func() {
		var (
			fake      *FakePrincipalAPI
			principal cloud.PrincipalProvider
		)

		BeforeEach(func() {
			fake = &FakePrincipalAPI{}
			principal = cfg.NewPrincipal(fake)
		})

		It("returns valid for a correct principal spec", func() {
			prncpl := newPrincipalCR()

			result := principal.ValidatePrincipalSpec(prncpl)
			Expect(result.Valid).To(BeTrue())
		})

		It("returns valid when spec.name is used", func() {
			prncpl := newPrincipalCR(func(b *vedro.CloudPrincipal) {
				b.Spec.Name = "actual-principal-name"
			})

			result := principal.ValidatePrincipalSpec(prncpl)
			Expect(result.Valid).To(BeTrue())
		})

		It("returns an error when spec.name is changed after creation", func() {
			prncpl := newPrincipalCR(func(b *vedro.CloudPrincipal) {
				b.Spec.Name = "new-name"
				b.Status.ExternalName = "old-name"
			})

			result := principal.ValidatePrincipalSpec(prncpl)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("spec.name cannot be changed"))
		})

		It("returns an error when metadata.name is used after spec.name was used", func() {
			prncpl := newPrincipalCR(func(b *vedro.CloudPrincipal) {
				b.Spec.Name = ""
				b.Status.ExternalName = "old-spec-name"
			})

			result := principal.ValidatePrincipalSpec(prncpl)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("metadata.name cannot be used"))
		})

	})

	return true
}
