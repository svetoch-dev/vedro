package helpers

import (
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
)

func BucketNameFromCR(bckt vedrov1alpha1.Bucket) string {
	bucketName := bckt.Name

	if bckt.Spec.Name != "" {
		bucketName = bckt.Spec.Name
	}

	return bucketName
}
