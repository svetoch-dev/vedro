package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/svetoch-dev/vedro/internal/cloud"
	iam "google.golang.org/api/iam/v1"
)

func saEmailAndFullName(name, projectId string) (string, string) {
	email := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", name, projectId)
	fullName := fmt.Sprintf(
		"projects/%s/serviceAccounts/%s",
		projectId,
		email,
	)

	return email, fullName
}

func saCreateKey(
	ctx context.Context,
	iamService *iam.Service,
	fullSaName string,
) (*iam.ServiceAccountKey, error) {
	key, err := iamService.
		Projects.
		ServiceAccounts.
		Keys.Create(fullSaName, &iam.CreateServiceAccountKeyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("create service account key: %w", err)
	}
	// Need to sleep because it takes some time
	// for the key to be available via gcp api
	time.Sleep(time.Second * 5)

	return key, nil
}

func saGetKey(
	ctx context.Context,
	iamService *iam.Service,
	keyId string,
) (*iam.ServiceAccountKey, error) {
	key, err := iamService.Projects.ServiceAccounts.Keys.Get(keyId).
		Context(ctx).
		Do()
	if err != nil {
		if isGoogleAPINotFound(err) {
			return nil, cloud.ErrAuthNotFound
		}

		return nil, fmt.Errorf("get service account key: %w", err)
	}

	return key, nil
}

func saDeleteKey(
	ctx context.Context,
	iamService *iam.Service,
	keyId string,
) error {
	if _, err := iamService.
		Projects.
		ServiceAccounts.
		Keys.
		Delete(keyId).
		Context(ctx).
		Do(); err != nil {
		if isGoogleAPINotFound(err) {
			return nil
		}
		return fmt.Errorf("delete service account key: %w", err)
	}

	return nil
}

func hasServiceAccountIAMBinding(
	ctx context.Context,
	service *iam.Service,
	serviceAccountEmail string,
	role string,
	principal string,
) (bool, error) {
	resource := fmt.Sprintf(
		"projects/-/serviceAccounts/%s",
		serviceAccountEmail,
	)

	policy, err := service.Projects.ServiceAccounts.
		GetIamPolicy(resource).
		Context(ctx).
		Do()
	if err != nil {
		return false, fmt.Errorf(
			"get IAM policy for service account %q: %w",
			serviceAccountEmail,
			err,
		)
	}

	for _, binding := range policy.Bindings {
		if binding.Role != role {
			continue
		}

		for _, member := range binding.Members {
			if member == principal {
				return true, nil
			}
		}
	}

	return false, nil
}

func grantServiceAccountIAMBinding(
	ctx context.Context,
	service *iam.Service,
	serviceAccountEmail string,
	role string,
	principal string,
) error {
	resource := fmt.Sprintf(
		"projects/-/serviceAccounts/%s",
		serviceAccountEmail,
	)

	policy, err := service.Projects.ServiceAccounts.
		GetIamPolicy(resource).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf(
			"get IAM policy for service account %q: %w",
			serviceAccountEmail,
			err,
		)
	}

	for _, binding := range policy.Bindings {
		if binding.Role != role {
			continue
		}

		for _, member := range binding.Members {
			if member == principal {
				return nil
			}
		}

		binding.Members = append(binding.Members, principal)

		_, err = service.Projects.ServiceAccounts.
			SetIamPolicy(
				resource,
				&iam.SetIamPolicyRequest{
					Policy: policy,
				},
			).
			Context(ctx).
			Do()

		if err != nil {
			return fmt.Errorf("set service account IAM policy: %w", err)
		}

		return nil
	}

	// Role doesn't exist yet.
	policy.Bindings = append(
		policy.Bindings,
		&iam.Binding{
			Role:    role,
			Members: []string{principal},
		},
	)

	_, err = service.Projects.ServiceAccounts.
		SetIamPolicy(
			resource,
			&iam.SetIamPolicyRequest{
				Policy: policy,
			},
		).
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("set service account IAM policy: %w", err)
	}

	return nil
}

func revokeServiceAccountIAMBinding(
	ctx context.Context,
	service *iam.Service,
	serviceAccountEmail string,
	role string,
	principal string,
) error {
	resource := fmt.Sprintf(
		"projects/-/serviceAccounts/%s",
		serviceAccountEmail,
	)

	policy, err := service.Projects.ServiceAccounts.
		GetIamPolicy(resource).
		Context(ctx).
		Do()

	if err != nil {
		if isGoogleAPINotFound(err) {
			return nil
		}
		return fmt.Errorf(
			"get IAM policy for service account %q: %w",
			serviceAccountEmail,
			err,
		)
	}

	hasChanges := false
	bindings := make([]*iam.Binding, 0, len(policy.Bindings))

	for _, binding := range policy.Bindings {
		if binding.Role != role {
			bindings = append(bindings, binding)
			continue
		}

		members := make([]string, 0, len(binding.Members))

		for _, member := range binding.Members {
			if member == principal {
				hasChanges = true
				continue
			}

			members = append(members, member)
		}

		// gcp sdk doesnt accept bindings with
		// 0 members so we pass it
		if len(members) == 0 {
			continue
		} else {
			binding.Members = members
			bindings = append(bindings, binding)
		}
	}

	policy.Bindings = bindings

	if hasChanges {
		_, err = service.Projects.ServiceAccounts.
			SetIamPolicy(
				resource,
				&iam.SetIamPolicyRequest{
					Policy: policy,
				},
			).
			Context(ctx).
			Do()

		if err != nil {
			return fmt.Errorf("set service account IAM policy: %w", err)
		}
	}

	return nil
}
