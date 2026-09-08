package usagepolicy

import (
	"testing"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testPolicy() vedro.UsagePolicySpec {
	return vedro.UsagePolicySpec{
		AllowedNamespaces: vedro.AllowedNamespacesSpec{Names: []string{"team"}},
		BucketPolicy:      vedro.BucketPolicySpec{AllowedNamePatterns: []string{"^team-.*$"}},
		PrincipalPolicy: vedro.PrincipalPolicySpec{
			AllowedNamePatterns:      []string{"^team-.*$"},
			AllowedReferencePatterns: []string{`^[^@]+@example\.com$`},
			AllowManaged:             true,
			AllowReferences:          true,
			AllowedKinds:             []vedro.PrincipalKind{vedro.PrincipalKindServiceAccount, vedro.PrincipalKindUser, vedro.PrincipalKindAllUsers},
		},
	}
}

func checkDecision(t *testing.T, got Decision, allowed bool) {
	t.Helper()
	if got.Allowed != allowed {
		t.Fatalf("Allowed = %v, want %v; message: %s", got.Allowed, allowed, got.Message)
	}
	if !got.Allowed && got.Message == "" {
		t.Error("restricted decision must explain the restriction")
	}
}

func TestCheckBucket(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*vedro.UsagePolicySpec, *vedro.Bucket)
		allowed bool
	}{
		{"metadata name matches", nil, true},
		{"explicit name matches", func(_ *vedro.UsagePolicySpec, b *vedro.Bucket) { b.Name = "unrelated"; b.Spec.Name = "team-data" }, true},
		{"explicit name overrides allowed metadata name", func(_ *vedro.UsagePolicySpec, b *vedro.Bucket) { b.Spec.Name = "other-data" }, false},
		{"name does not match", func(_ *vedro.UsagePolicySpec, b *vedro.Bucket) { b.Name = "other-data" }, false},
		{"later pattern matches", func(p *vedro.UsagePolicySpec, _ *vedro.Bucket) {
			p.BucketPolicy.AllowedNamePatterns = []string{"^other-.*$", "^team-.*$"}
		}, true},
		{"empty patterns deny", func(p *vedro.UsagePolicySpec, _ *vedro.Bucket) { p.BucketPolicy.AllowedNamePatterns = nil }, false},
		{"unlisted namespace denied", func(_ *vedro.UsagePolicySpec, b *vedro.Bucket) { b.Namespace = "other" }, false},
		{"all namespaces allowed", func(p *vedro.UsagePolicySpec, b *vedro.Bucket) {
			p.AllowedNamespaces = vedro.AllowedNamespacesSpec{All: true}
			b.Namespace = "other"
		}, true},
		{"empty namespace list denies", func(p *vedro.UsagePolicySpec, _ *vedro.Bucket) { p.AllowedNamespaces.Names = nil }, false},
		{"active resource uses spec name", func(_ *vedro.UsagePolicySpec, b *vedro.Bucket) { b.Status.ExternalName = "other-data" }, true},
		{"deletion denies forbidden recorded name", func(_ *vedro.UsagePolicySpec, b *vedro.Bucket) {
			now := metav1.Now()
			b.DeletionTimestamp = &now
			b.Status.ExternalName = "other-data"
		}, false},
		{"deletion allows recorded name despite spec change", func(_ *vedro.UsagePolicySpec, b *vedro.Bucket) {
			now := metav1.Now()
			b.DeletionTimestamp = &now
			b.Status.ExternalName = "team-data"
			b.Spec.Name = "other-data"
		}, true},
		{"deletion without recorded name checks spec", func(_ *vedro.UsagePolicySpec, b *vedro.Bucket) {
			now := metav1.Now()
			b.DeletionTimestamp = &now
			b.Spec.Name = "other-data"
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := testPolicy()
			bucket := vedro.Bucket{ObjectMeta: metav1.ObjectMeta{Name: "team-data", Namespace: "team"}}
			if tt.change != nil {
				tt.change(&policy, &bucket)
			}
			checkDecision(t, CheckBucket(policy, bucket), tt.allowed)
		})
	}
}

func TestCheckManagedPrincipal(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*vedro.UsagePolicySpec, *vedro.CloudPrincipal)
		allowed bool
	}{
		{"managed name matches", nil, true},
		{"managed name denied", func(_ *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) { p.Spec.Managed.Name = "other-account" }, false},
		{"managed disabled", func(p *vedro.UsagePolicySpec, _ *vedro.CloudPrincipal) { p.PrincipalPolicy.AllowManaged = false }, false},
		{"references disabled does not block managed", func(p *vedro.UsagePolicySpec, _ *vedro.CloudPrincipal) { p.PrincipalPolicy.AllowReferences = false }, true},
		{"empty name patterns deny", func(p *vedro.UsagePolicySpec, _ *vedro.CloudPrincipal) { p.PrincipalPolicy.AllowedNamePatterns = nil }, false},
		{"kind denied", func(p *vedro.UsagePolicySpec, _ *vedro.CloudPrincipal) {
			p.PrincipalPolicy.AllowedKinds = []vedro.PrincipalKind{vedro.PrincipalKindUser}
		}, false},
		{"empty kinds deny", func(p *vedro.UsagePolicySpec, _ *vedro.CloudPrincipal) { p.PrincipalPolicy.AllowedKinds = nil }, false},
		{"namespace denied", func(_ *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) { p.Namespace = "other" }, false},
		{"active principal uses spec kind", func(_ *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) { p.Status.Kind = vedro.PrincipalKindRole }, true},
		{"deletion checks recorded kind", func(policy *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			now := metav1.Now()
			p.DeletionTimestamp = &now
			p.Spec.Kind = vedro.PrincipalKindUser
			policy.PrincipalPolicy.AllowedKinds = []vedro.PrincipalKind{vedro.PrincipalKindUser}
		}, false},
		{"deletion allows recorded kind despite spec change", func(policy *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			now := metav1.Now()
			p.DeletionTimestamp = &now
			p.Spec.Kind = vedro.PrincipalKindUser
			policy.PrincipalPolicy.AllowedKinds = []vedro.PrincipalKind{vedro.PrincipalKindServiceAccount}
		}, true},
		{"deletion checks recorded name", func(_ *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			now := metav1.Now()
			p.DeletionTimestamp = &now
			p.Status.ExternalName = "other-account"
		}, false},
		{"deletion ignores changed spec name", func(_ *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			now := metav1.Now()
			p.DeletionTimestamp = &now
			p.Spec.Managed.Name = "other-account"
		}, true},
		{"unknown deletion kind fails closed", func(_ *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			now := metav1.Now()
			p.DeletionTimestamp = &now
			p.Status.Kind = ""
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := testPolicy()
			principal := vedro.CloudPrincipal{
				ObjectMeta: metav1.ObjectMeta{Name: "unrelated-cr-name", Namespace: "team"},
				Spec:       vedro.CloudPrincipalSpec{Kind: vedro.PrincipalKindServiceAccount, ManagementPolicy: vedro.PrincipalManagementPolicyManaged, Managed: &vedro.ManagedPrincipalSpec{Name: "team-account"}},
				Status:     vedro.CloudPrincipalStatus{Kind: vedro.PrincipalKindServiceAccount, ExternalName: "team-account"},
			}
			if tt.change != nil {
				tt.change(&policy, &principal)
			}
			checkDecision(t, CheckPrincipal(policy, principal), tt.allowed)
		})
	}
}

func TestCheckReferencedPrincipal(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*vedro.UsagePolicySpec, *vedro.CloudPrincipal)
		allowed bool
	}{
		{"reference matches separate reference patterns", nil, true},
		{"reference outside domain denied", func(_ *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) { p.Spec.Reference.Name = "person@other.com" }, false},
		{"references disabled", func(p *vedro.UsagePolicySpec, _ *vedro.CloudPrincipal) { p.PrincipalPolicy.AllowReferences = false }, false},
		{"managed disabled does not block references", func(p *vedro.UsagePolicySpec, _ *vedro.CloudPrincipal) { p.PrincipalPolicy.AllowManaged = false }, true},
		{"empty reference patterns deny", func(p *vedro.UsagePolicySpec, _ *vedro.CloudPrincipal) {
			p.PrincipalPolicy.AllowedReferencePatterns = nil
		}, false},
		{"all users needs no reference or patterns", func(policy *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			p.Spec.Kind = vedro.PrincipalKindAllUsers
			p.Spec.Reference = nil
			policy.PrincipalPolicy.AllowedReferencePatterns = nil
		}, true},
		{"all users still requires allowed kind", func(policy *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			p.Spec.Kind = vedro.PrincipalKindAllUsers
			p.Spec.Reference = nil
			policy.PrincipalPolicy.AllowedKinds = []vedro.PrincipalKind{vedro.PrincipalKindUser}
		}, false},
		{"all users still requires references enabled", func(policy *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			p.Spec.Kind = vedro.PrincipalKindAllUsers
			p.Spec.Reference = nil
			policy.PrincipalPolicy.AllowReferences = false
		}, false},
		{"all users still requires allowed namespace", func(_ *vedro.UsagePolicySpec, p *vedro.CloudPrincipal) {
			p.Spec.Kind = vedro.PrincipalKindAllUsers
			p.Spec.Reference = nil
			p.Namespace = "other"
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := testPolicy()
			principal := vedro.CloudPrincipal{
				ObjectMeta: metav1.ObjectMeta{Name: "team-reference", Namespace: "team"},
				Spec:       vedro.CloudPrincipalSpec{Kind: vedro.PrincipalKindUser, ManagementPolicy: vedro.PrincipalManagementPolicyReference, Reference: &vedro.ReferencedPrincipalSpec{Name: "person@example.com"}},
			}
			if tt.change != nil {
				tt.change(&policy, &principal)
			}
			checkDecision(t, CheckPrincipal(policy, principal), tt.allowed)
		})
	}
}
