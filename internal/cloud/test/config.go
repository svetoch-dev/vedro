package cloudtest

import (
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

// Config parameterizes the shared bucket lifecycle specs with the details
// that differ between providers.
type Config struct {
	// Location is a location the provider accepts in spec.location.
	Location string
	// NormalizedLocation is how the provider stores/returns Location
	// (e.g. GCP upper-cases it: "europe-west1" -> "EUROPE-WEST1").
	NormalizedLocation string

	// OtherLocation is a different valid location, used to test the
	// "bucket already exists in another location" case.
	OtherLocation string
	// OtherNormalizedLocation is the normalized form of OtherLocation.
	OtherNormalizedLocation string

	ProviderConfigType vedro.ProviderType

	BucketPropertiesMods []func(*vedro.BucketProperties)

	// Bucket provider capabilities
	BucketCaps cloud.BucketCapabilities

	// NewBucket wires the provider's cloud.BucketProvider to the supplied
	// fake API. Implemented inside each provider's package so it can reach
	// unexported fields.
	NewBucket func(api cloud.BucketAPI) cloud.BucketProvider

	// NewPrincipal wires the provider's cloud.PrincipalProvider to the supplied
	// fake API. Implemented inside each provider's package so it can reach
	// unexported fields.
	NewPrincipal func(api cloud.PrincipalAPI) cloud.PrincipalProvider
}
