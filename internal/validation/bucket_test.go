package validation

import (
	"strings"
	"testing"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValid(t *testing.T) {
	result := Valid()
	if !result.Valid {
		t.Errorf("Expect Valid() to return Valid=true, got false")
	}
	if result.Message != "" {
		t.Errorf("Expect Valid() message to be empty, got %q", result.Message)
	}
}

func TestInvalid(t *testing.T) {
	result := Invalid("something went wrong")
	if result.Valid {
		t.Errorf("Expect Invalid() to return Valid=false, got true")
	}
	if result.Message != "something went wrong" {
		t.Errorf("Expect Invalid() message %q, got %q", "something went wrong", result.Message)
	}
}

func newBucket(name, specName, externalName string) vedro.Bucket {
	return vedro.Bucket{
		Spec: vedro.BucketSpec{
			Name: specName,
		},
		Status: vedro.BucketStatus{
			ExternalName: externalName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func TestValidateBucketNameImmutability(t *testing.T) {
	tests := []struct {
		name    string
		bucket  vedro.Bucket
		valid   bool
		message string
	}{
		{
			name:   "no external name set",
			bucket: newBucket("my-bucket", "", ""),
			valid:  true,
		},
		{
			name:   "spec.name matches external name",
			bucket: newBucket("cr-name", "same-name", "same-name"),
			valid:  true,
		},
		{
			name:   "metadata.name matches external name",
			bucket: newBucket("my-bucket", "", "my-bucket"),
			valid:  true,
		},
		{
			name:    "spec.name changed after creation",
			bucket:  newBucket("cr-name", "new-name", "old-name"),
			valid:   false,
			message: "spec.name cannot be changed after bucket creation",
		},
		{
			name:    "metadata.name used after spec.name was used",
			bucket:  newBucket("cr-name", "", "old-spec-name"),
			valid:   false,
			message: "metadata.name cannot be used as the bucket name source if spec.Name was used and bucket is created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateBucketNameImmutability(tt.bucket)
			if result.Valid != tt.valid {
				t.Errorf("Expect Valid=%v, got %v", tt.valid, result.Valid)
			}
			if !tt.valid && !strings.Contains(result.Message, tt.message) {
				t.Errorf("Expect message to contain %q, got %q", tt.message, result.Message)
			}
		})
	}
}

func TestValidateBucketLocation(t *testing.T) {
	acceptAll := func(string) *ValidationResult {
		v := Valid()
		return &v
	}
	rejectAll := func(string) *ValidationResult {
		v := Invalid("provider rejected location")
		return &v
	}
	deferToGeneric := func(string) *ValidationResult {
		return nil
	}

	tests := []struct {
		name     string
		location string
		fn       func(string) *ValidationResult
		valid    bool
		message  string
	}{
		{
			name:     "empty location",
			location: "",
			fn:       deferToGeneric,
			valid:    false,
			message:  "location is an empty string",
		},
		{
			name:     "provider accepts location",
			location: "anywhere",
			fn:       acceptAll,
			valid:    true,
		},
		{
			name:     "provider rejects location",
			location: "somewhere",
			fn:       rejectAll,
			valid:    false,
			message:  "provider rejected location",
		},
		{
			name:     "provider defers to generic valid regional",
			location: "europe-west1",
			fn:       deferToGeneric,
			valid:    true,
		},
		{
			name:     "provider defers to generic unsupported location",
			location: "Somewhere",
			fn:       deferToGeneric,
			valid:    false,
			message:  "unsupported bucket location",
		},
		{
			name:     "provider defers to generic location with invalid characters",
			location: "europe_west1",
			fn:       deferToGeneric,
			valid:    false,
			message:  "unsupported bucket location",
		},
		{
			name:     "provider defers to generic multi-word unsupported location",
			location: "not a region",
			fn:       deferToGeneric,
			valid:    false,
			message:  "unsupported bucket location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateBucketLocation(tt.location, tt.fn)
			if result.Valid != tt.valid {
				t.Errorf("Expect Valid=%v, got %v", tt.valid, result.Valid)
			}
			if !tt.valid && !strings.Contains(result.Message, tt.message) {
				t.Errorf("Expect message to contain %q, got %q", tt.message, result.Message)
			}
		})
	}
}

func TestValidateBucketName(t *testing.T) {
	deferToGeneric := func(string) *ValidationResult {
		return nil
	}
	providerRejects := func(string) *ValidationResult {
		v := Invalid("provider rejected name")
		return &v
	}
	providerAccepts := func(string) *ValidationResult {
		v := Valid()
		return &v
	}

	tests := []struct {
		name    string
		input   string
		fn      func(string) *ValidationResult
		valid   bool
		message string
	}{
		{
			name:    "empty name",
			input:   "",
			fn:      deferToGeneric,
			valid:   false,
			message: "name is an empty string",
		},
		{
			name:    "provider rejects name",
			input:   "valid-name",
			fn:      providerRejects,
			valid:   false,
			message: "provider rejected name",
		},
		{
			name:    "provider accepts name",
			input:   "valid-name",
			fn:      providerAccepts,
			valid:   true,
			message: "provider accepted name",
		},

		{
			name:  "valid name",
			input: "valid-name",
			fn:    deferToGeneric,
			valid: true,
		},
		{
			name:  "name with dots and underscores",
			input: "my.bucket_name",
			fn:    deferToGeneric,
			valid: true,
		},
		{
			name:    "name too short",
			input:   "ab",
			fn:      deferToGeneric,
			valid:   false,
			message: "bucket name must be 3-63 characters",
		},
		{
			name:    "name with uppercase letters",
			input:   "My-Bucket",
			fn:      deferToGeneric,
			valid:   false,
			message: "bucket name must be 3-63 characters",
		},
		{
			name:    "name with invalid characters",
			input:   "my!bucket",
			fn:      deferToGeneric,
			valid:   false,
			message: "bucket name must be 3-63 characters",
		},
		{
			name:    "name starts with dot",
			input:   ".mybucket",
			fn:      deferToGeneric,
			valid:   false,
			message: "bucket name must be 3-63 characters",
		},
		{
			name:    "name ends with dash",
			input:   "mybucket-",
			fn:      deferToGeneric,
			valid:   false,
			message: "bucket name must be 3-63 characters",
		},
		{
			name:    "name with consecutive dots",
			input:   "my..bucket",
			fn:      deferToGeneric,
			valid:   false,
			message: "bucket name must not contain consecutive dots",
		},
		{
			name:    "name with dot next to dash",
			input:   "my.-bucket",
			fn:      deferToGeneric,
			valid:   false,
			message: "bucket name must not contain dots next to dashes",
		},
		{
			name:    "name with dash next to dot",
			input:   "my-.bucket",
			fn:      deferToGeneric,
			valid:   false,
			message: "bucket name must not contain dots next to dashes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateBucketName(tt.input, tt.fn)
			if result.Valid != tt.valid {
				t.Errorf("Expect Valid=%v, got %v", tt.valid, result.Valid)
			}
			if !tt.valid && !strings.Contains(result.Message, tt.message) {
				t.Errorf("Expect message to contain %q, got %q", tt.message, result.Message)
			}
		})
	}
}
