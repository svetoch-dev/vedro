package gcp

import (
	"context"
	"errors"
	"fmt"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

type BucketAccess struct {
	api cloud.BucketAPI
}

func (b *BucketAccess) EnsureBucketAccess(
	ctx context.Context,
	bucket vedro.Bucket,
	principal vedro.CloudPrincipal,
	access vedro.BucketAccess,
) (*cloud.BucketAccessAttrs, error) {
	spec := access.Spec
	status := access.Status

	got := status.Applied

	want := &cloud.BucketAccessAttrs{
		BucketName:    bucket.Status.ExternalName,
		BucketId:      bucket.Status.ExternalId,
		PrincipalId:   fmt.Sprintf("serviceAccount:%s", principal.Status.ExternalId),
		GrantedAccess: spec.Access.Level,
	}

	if got != nil && *got != vedro.BucketAccessProperties(*want) {
		gotAttrs := cloud.BucketAccessAttrs(*got)
		hasAccess, err := b.api.HasAccess(ctx, gotAttrs)

		if err != nil && !errors.Is(err, cloud.ErrBucketNotFound) {
			return nil, fmt.Errorf("Has access check failed: %w", err)
		}

		if hasAccess {
			revokeErr := b.api.RevokeAccess(ctx, gotAttrs)
			if revokeErr != nil {
				return nil, fmt.Errorf("Revoke access from principal failed: %w", revokeErr)
			}
		}
	}

	hasAccess, err := b.api.HasAccess(ctx, *want)
	if err != nil {
		return nil, fmt.Errorf("Has access check failed: %w", err)
	}

	if !hasAccess {
		grantErr := b.api.GrantAccess(ctx, *want)
		if grantErr != nil {
			return nil, fmt.Errorf("Granting access to principal failed: %w", grantErr)
		}
	}

	return want, nil

}

func (b *BucketAccess) DeleteBucketAccess(ctx context.Context, access vedro.BucketAccess) error {
	if access.Status.Applied == nil {
		return nil
	}
	applied := cloud.BucketAccessAttrs(*access.Status.Applied)
	hasAccess, err := b.api.HasAccess(ctx, applied)
	if err != nil {
		if errors.Is(err, cloud.ErrBucketNotFound) {
			return nil
		}
		return fmt.Errorf("Has access check failed: %w", err)
	}

	if hasAccess {
		revokeErr := b.api.RevokeAccess(ctx, applied)
		if revokeErr != nil {
			if errors.Is(revokeErr, cloud.ErrBucketNotFound) {
				return nil
			}
			return fmt.Errorf("Revoke access from principal failed: %w", revokeErr)
		}
	}

	return nil
}
