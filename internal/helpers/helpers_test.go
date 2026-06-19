package helpers

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
)

func TestBucketNameFromCR(t *testing.T) {
	tests := []struct {
		name     string
		bucket   vedrov1alpha1.Bucket
		expected string
	}{
		{
			name: "returns metadata.name when spec.name is empty",
			bucket: vedrov1alpha1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "cr-name"},
			},
			expected: "cr-name",
		},
		{
			name: "returns spec.name when set",
			bucket: vedrov1alpha1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "cr-name"},
				Spec:       vedrov1alpha1.BucketSpec{Name: "actual-bucket"},
			},
			expected: "actual-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BucketNameFromCR(tt.bucket)
			if got != tt.expected {
				t.Errorf("BucketNameFromCR() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPtr(t *testing.T) {
	t.Run("int pointer", func(t *testing.T) {
		p := Ptr(42)
		if p == nil || *p != 42 {
			t.Errorf("Ptr(42) = %v, want pointer to 42", p)
		}
	})

	t.Run("string pointer", func(t *testing.T) {
		p := Ptr("hello")
		if p == nil || *p != "hello" {
			t.Errorf("Ptr(\"hello\") = %v, want pointer to \"hello\"", p)
		}
	})

	t.Run("bool pointer", func(t *testing.T) {
		p := Ptr(true)
		if p == nil || *p != true {
			t.Errorf("Ptr(true) = %v, want pointer to true", p)
		}
	})
}

func TestGetSecretData(t *testing.T) {
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"access-key": []byte("access-value"),
			"secret-key": []byte("secret-value"),
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(secret).Build()

	t.Run("returns requested keys", func(t *testing.T) {
		data, err := GetSecretData(ctx, kubeClient, corev1.SecretReference{
			Name:      "my-secret",
			Namespace: "default",
		}, "access-key", "secret-key")

		if err != nil {
			t.Fatalf("GetSecretData() unexpected error: %v", err)
		}

		if string(data["access-key"]) != "access-value" {
			t.Errorf("access-key = %q, want %q", data["access-key"], "access-value")
		}
		if string(data["secret-key"]) != "secret-value" {
			t.Errorf("secret-key = %q, want %q", data["secret-key"], "secret-value")
		}
	})

	t.Run("returns error when secret is not found", func(t *testing.T) {
		_, err := GetSecretData(ctx, kubeClient, corev1.SecretReference{
			Name:      "missing-secret",
			Namespace: "default",
		}, "access-key")

		if err == nil {
			t.Fatal("GetSecretData() expected error, got nil")
		}
	})

	t.Run("returns error when key is missing", func(t *testing.T) {
		_, err := GetSecretData(ctx, kubeClient, corev1.SecretReference{
			Name:      "my-secret",
			Namespace: "default",
		}, "missing-key")

		if err == nil {
			t.Fatal("GetSecretData() expected error, got nil")
		}
	})
}

func TestGetSecretDataEmptyKeys(t *testing.T) {
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(secret).Build()

	data, err := GetSecretData(ctx, kubeClient, corev1.SecretReference{
		Name:      "my-secret",
		Namespace: "default",
	})

	if err != nil {
		t.Fatalf("GetSecretData() unexpected error: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("GetSecretData() returned %d entries, want 0", len(data))
	}
}

var errAlwaysFail = errors.New("always fail")

type failingClient struct {
	client.Client
}

func (f *failingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return errAlwaysFail
}

func TestGetSecretDataClientError(t *testing.T) {
	ctx := context.Background()

	_, err := GetSecretData(ctx, &failingClient{}, corev1.SecretReference{
		Name: "my-secret",
	})

	if err == nil {
		t.Fatal("GetSecretData() expected error, got nil")
	}

	if !errors.Is(err, errAlwaysFail) {
		t.Errorf("GetSecretData() error = %v, want to wrap %v", err, errAlwaysFail)
	}
}
