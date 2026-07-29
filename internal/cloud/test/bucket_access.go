package cloudtest

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

// BucketAccessProviderTests registers the provider-agnostic EnsureBucketAccess and
// DeleteBucketAccess specs. Call it from each provider's Ginkgo suite, e.g.:
//
//	var _ = cloudtest.BucketAccessProviderTests(cloudtest.Config{...})
func BucketAccessProviderTests(cfg Config) bool {
	newBucketAccessCR := func(mods ...func(*vedro.BucketAccess)) vedro.BucketAccess {
		return NewBucketAccessCR("bucket-access-this", mods...)
	}
	Describe("BucketAccessProvider.EnsureBucketAccess", func() {
		var (
			ctx          context.Context
			fake         *FakeBucketAPI
			bucketAccess cloud.BucketAccessProvider
		)

		BeforeEach(func() {
			ctx = context.Background()
			fake = &FakeBucketAPI{}
			bucketAccess = cfg.NewBucketAccess(fake)
		})

		It("grants access if it does not exist", func() {
			fake.HasAccessErr = nil
			fake.HasAccessResult = false
			fake.GrantAccessErr = nil
			bucket := NewBucketCR("test-bucket", cfg.Location, func(b *vedro.Bucket) {
				b.Status.ExternalName = "test-bucket"
				b.Status.ExternalId = "someid"
			})
			principal := NewPrincipalCR("test-principal", func(p *vedro.CloudPrincipal) {
				p.Status.ExternalId = "someid"
			})
			access := newBucketAccessCR()
			attrs, err := bucketAccess.EnsureBucketAccess(ctx, bucket, principal, access)

			Expect(err).NotTo(HaveOccurred())
			Expect(fake.HasAccessCalls()).To(Equal(1))
			Expect(fake.GrantAccessCalls()).To(Equal(1))
			Expect(*attrs).To(Equal(cloud.BucketAccessAttrs{
				BucketName:    "test-bucket",
				BucketId:      "someid",
				PrincipalId:   "someid",
				GrantedAccess: vedro.ObjectReader,
			}))
		})
		It("errors if HasAccess has error", func() {
			fake.HasAccessErr = errors.New("some error")
			fake.GrantAccessErr = nil
			bucket := NewBucketCR("test-bucket", cfg.Location, func(b *vedro.Bucket) {
				b.Status.ExternalName = "test-bucket"
				b.Status.ExternalId = "someid"
			})
			principal := NewPrincipalCR("test-principal", func(p *vedro.CloudPrincipal) {
				p.Status.ExternalId = "someid"
			})
			access := newBucketAccessCR()
			attrs, err := bucketAccess.EnsureBucketAccess(ctx, bucket, principal, access)

			Expect(err).To(HaveOccurred())
			Expect(fake.HasAccessCalls()).To(Equal(1))
			Expect(fake.GrantAccessCalls()).To(Equal(0))
			Expect(attrs).To(BeNil())
		})
		It("errors if GrantError has error", func() {
			fake.HasAccessResult = false
			fake.HasAccessErr = nil
			fake.GrantAccessErr = errors.New("some error")
			bucket := NewBucketCR("test-bucket", cfg.Location, func(b *vedro.Bucket) {
				b.Status.ExternalName = "test-bucket"
				b.Status.ExternalId = "someid"
			})
			principal := NewPrincipalCR("test-principal", func(p *vedro.CloudPrincipal) {
				p.Status.ExternalId = "someid"
			})
			access := newBucketAccessCR()
			attrs, err := bucketAccess.EnsureBucketAccess(ctx, bucket, principal, access)

			Expect(err).To(HaveOccurred())
			Expect(fake.HasAccessCalls()).To(Equal(1))
			Expect(fake.GrantAccessCalls()).To(Equal(1))
			Expect(attrs).To(BeNil())
		})
		It("skips GrantAccess if access already exists", func() {
			fake.HasAccessResult = true
			fake.HasAccessErr = nil
			fake.GrantAccessErr = nil
			bucket := NewBucketCR("test-bucket", cfg.Location, func(b *vedro.Bucket) {
				b.Status.ExternalName = "test-bucket"
				b.Status.ExternalId = "someid"
			})
			principal := NewPrincipalCR("test-principal", func(p *vedro.CloudPrincipal) {
				p.Status.ExternalId = "someid"
			})
			access := newBucketAccessCR()
			attrs, err := bucketAccess.EnsureBucketAccess(ctx, bucket, principal, access)

			Expect(err).NotTo(HaveOccurred())
			Expect(fake.HasAccessCalls()).To(Equal(1))
			Expect(fake.GrantAccessCalls()).To(Equal(0))
			Expect(*attrs).To(Equal(cloud.BucketAccessAttrs{
				BucketName:    "test-bucket",
				BucketId:      "someid",
				PrincipalId:   "someid",
				GrantedAccess: vedro.ObjectReader,
			}))
		})
		It("Does not revoke if applied and CR spec do not match", func() {
			fake.HasAccessResult = true
			fake.HasAccessErr = nil
			fake.GrantAccessErr = nil
			fake.RevokeAccessErr = nil
			bucket := NewBucketCR("test-bucket", cfg.Location, func(b *vedro.Bucket) {
				b.Status.ExternalName = "test-bucket"
				b.Status.ExternalId = "someid"
			})
			principal := NewPrincipalCR("test-principal", func(p *vedro.CloudPrincipal) {
				p.Status.ExternalId = "someid"
			})
			access := newBucketAccessCR(func(ba *vedro.BucketAccess) {
				ba.Status.Applied = &vedro.BucketAccessProperties{
					BucketName:    "test-bucket",
					BucketId:      "someid",
					PrincipalId:   "someid",
					GrantedAccess: vedro.ObjectReader,
				}
			})
			attrs, err := bucketAccess.EnsureBucketAccess(ctx, bucket, principal, access)

			Expect(err).NotTo(HaveOccurred())
			Expect(fake.HasAccessCalls()).To(Equal(1))
			Expect(fake.GrantAccessCalls()).To(Equal(0))
			Expect(fake.RevokeAccessCalls()).To(Equal(0))
			Expect(*attrs).To(Equal(cloud.BucketAccessAttrs{
				BucketName:    "test-bucket",
				BucketId:      "someid",
				PrincipalId:   "someid",
				GrantedAccess: vedro.ObjectReader,
			}))
		})
		It("RevokesAccess if applied and CR spec match and principal HasAccess", func() {
			fake.HasAccessResult = true
			fake.HasAccessErr = nil
			fake.GrantAccessErr = nil
			fake.RevokeAccessErr = nil
			bucket := NewBucketCR("test-bucket", cfg.Location, func(b *vedro.Bucket) {
				b.Status.ExternalName = "test-bucket"
				b.Status.ExternalId = "someid"
			})
			principal := NewPrincipalCR("test-principal", func(p *vedro.CloudPrincipal) {
				p.Status.ExternalId = "someid"
			})
			access := newBucketAccessCR(func(ba *vedro.BucketAccess) {
				ba.Status.Applied = &vedro.BucketAccessProperties{
					BucketName:    "test-bucket",
					BucketId:      "someid",
					PrincipalId:   "someid",
					GrantedAccess: vedro.ObjectReader,
				}
				ba.Spec.Access.Level = vedro.BucketAdmin
			})
			attrs, err := bucketAccess.EnsureBucketAccess(ctx, bucket, principal, access)

			Expect(err).NotTo(HaveOccurred())
			Expect(fake.HasAccessCalls()).To(Equal(2))
			Expect(fake.RevokeAccessCalls()).To(Equal(1))
			Expect(fake.GrantAccessCalls()).To(Equal(1))
			Expect(*attrs).To(Equal(cloud.BucketAccessAttrs{
				BucketName:    "test-bucket",
				BucketId:      "someid",
				PrincipalId:   "someid",
				GrantedAccess: vedro.BucketAdmin,
			}))
		})

	})
	return true
}
