package gcp

import (
	"maps"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

func appliedState(
	location string,
	bckt vedro.Bucket,
) *cloud.BucketAttrs {
	spec := bckt.Spec
	bucketName := helpers.BucketNameFromCR(bckt)

	return &cloud.BucketAttrs{
		Name:     bucketName,
		Location: location,
		Properties: &vedro.BucketProperties{
			StorageClass:           spec.StorageClass,
			Labels:                 maps.Clone(spec.Labels),
			Versioning:             normalizedBucketVersioning(spec.Versioning.DeepCopy()),
			PublicAccessPrevention: normalizedBucketPAP(helpers.CloneBool(spec.PublicAccessPrevention)),
			Lifecycle:              normalizedBucketLifecycle(spec.Lifecycle.DeepCopy()),
		},
	}
}

func normalizedBucketVersioning(ver *vedro.BucketVersioning) *vedro.BucketVersioning {
	if ver == nil {
		return &vedro.BucketVersioning{
			Enabled: false,
		}
	}
	return ver
}

func normalizedBucketPAP(pap *bool) *bool {
	if pap == nil {
		return helpers.Ptr(false)
	}

	return pap
}

func normalizedBucketLifecycle(
	lifecycle *vedro.BucketLifecycle,
) *vedro.BucketLifecycle {
	normalized := &vedro.BucketLifecycle{}
	if lifecycle == nil || len(lifecycle.Rules) == 0 {
		return normalized
	}

	for _, rule := range lifecycle.Rules {
		if !rule.Enabled {
			continue
		}
		normalized.Rules = append(normalized.Rules,
			vedro.BucketLifecycleRule{
				AgeDays: rule.AgeDays,
				Action:  rule.Action,
				Enabled: rule.Enabled,
			},
		)
	}
	return normalized
}

func normalizedCloudSpecific(cfg *vedro.BucketCloudSpecificConfig) *vedro.BucketCloudSpecificConfig {
	if cfg == nil || cfg.Gcp == nil || cfg.Gcp.SoftDeletePolicy == nil {
		return &vedro.BucketCloudSpecificConfig{
			Gcp: &vedro.BucketGcpConfig{
				SoftDeletePolicy: defaultSoftDelete,
			},
		}
	}

	return cfg
}
