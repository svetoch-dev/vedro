package conditions

// Condition types used across Vedro resources.
const (
	// TypeReady indicates whether a resource has been successfully reconciled
	// and is ready for use.
	TypeReady = "Ready"

	// TypeProviderConfigReady indicates whether the referenced ProviderConfig
	// is ready for use.
	TypeProviderConfigReady = "ProviderConfigReady"

	// TypeBucketReady indicates whether a Bucket dependency is ready for use.
	TypeBucketReady = "BucketReady"

	// TypeCloudPrincipalReady indicates whether a CloudPrincipal dependency is ready for use.
	TypeCloudPrincipalReady = "CloudPrincipalReady"
)

// Generic condition reasons

const (
	ReasonNoConditions        = "NoConditions"
	ReasonGenerationMissmatch = "GenerationMissmatch"
)

// Condition reasons for Bucket resources.
const (
	ReasonBucketNotFound            = "BucketNotFound"
	ReasonBucketGetFailed           = "BucketGetFailed"
	ReasonBucketUnsupportedFeatures = "BucketUnsupportedFeatures"
	ReasonBucketInvalidSpec         = "BucketInvalidSpec"
	ReasonBucketEnsureError         = "BucketEnsureError"
	ReasonBucketReconciled          = "Reconciled"
	ReasonBucketDeleteError         = "BucketDeleteError"
	ReasonBucketReady               = "BucketReady"
)

// Condition reasons for CloudPrincipal resources.
const (
	ReasonCloudPrincipalNotFound            = "CloudPrincipalNotFound"
	ReasonCloudPrincipalGetFailed           = "CloudPrincipalGetFailed"
	ReasonCloudPrincipalInvalidSpec         = "CloudPrincipalInvalidSpec"
	ReasonCloudPrincipalDeleteError         = "CloudPrincipalDeleteError"
	ReasonCloudPrincipalUnsupportedFeatures = "CloudPrincipalUnsupportedFeatures"
	ReasonCloudPrincipalEnsureError         = "CloudPrincipalEnsureError"
	ReasonCloudPrincipalReconciled          = "Reconciled"
	ReasonCloudPrincipalReady               = "CloudPrincipalReady"
)

// Condition reasons for ProviderConfig resources.
const (
	ReasonProviderConfigNotFound    = "ProviderConfigNotFound"
	ReasonProviderConfigGetFailed   = "ProviderConfigGetFailed"
	ReasonProviderConfigError       = "ProviderConfigError"
	ReasonProviderConfigInvalidSpec = "ProviderConfigInvalidSpec"
	ReasonProviderConfigReconciled  = "Reconciled"
	ReasonProviderConfigSet         = "ProviderConfigSet"
	ReasonProviderConfigNotReady    = "ProviderConfigNotReady"
	ReasonProviderConfigReady       = "ProviderConfigReady"
)

// Condition reasons for BucketAccess resources.
const (
	ReasonBucketAccessNotFound                = "BucketAccessNotFound"
	ReasonBucketAccessGetFailed               = "BucketAccessGetFailed"
	ReasonBucketAccessDependencyNotReady      = "BucketAccessDependencyNotReady"
	ReasonBucketAccessProviderConfigMissMatch = "BucketAccessProviderConfigMissMatch"
	ReasonBucketAccessDeleteError             = "BucketAccessDeleteError"
	ReasonBucketAccessUnsupportedFeatures     = "BucketAccessUnsupportedFeatures"
	ReasonBucketAccessEnsureError             = "BucketAccessEnsureError"
	ReasonBucketAccessReconciled              = "Reconciled"
)
