package v1alpha1

type ProviderType string

const (
	ProviderTypeGCP    ProviderType = "GCP"
	ProviderTypeYandex ProviderType = "Yandex"
)

type AuthMethod string

const (
	AuthMethodStaticCredentials AuthMethod = "StaticCredentials"
	AuthMethodWorkloadIdentity  AuthMethod = "WorkloadIdentity"
)
