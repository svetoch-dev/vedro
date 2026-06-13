package conditions

// Condition types used across Vedro resources.
const (
	// TypeReady is the standard Ready condition type for Bucket resources.
	TypeReady = "Ready"

	// TypeProviderConfigReady is the Ready condition type for ProviderConfig resources.
	TypeProviderConfigReady = "ProviderConfigReady"
)

// Condition reasons for Bucket resources.
const (
	ReasonBucketNotFound            = "BucketNotFound"
	ReasonBucketGetFailed           = "BucketGetFailed"
	ReasonBucketInvalidCapabilities = "BucketInvalidCapabilities"
	ReasonBucketReconciled          = "Reconciled"
)

// Condition reasons for ProviderConfig resources.
const (
	ReasonProviderConfigNotFound   = "ProviderConfigNotFound"
	ReasonProviderConfigGetFailed  = "ProviderConfigGetFailed"
	ReasonProviderConfigError      = "ProviderConfigError"
	ReasonProviderConfigReconciled = "Reconciled"
	ReasonProviderConfigSet        = "ProviderConfigSet"
)
