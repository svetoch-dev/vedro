package cloud

import (
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
)

type BucketState struct {
	ExternalName     string
	Location         string
	ObservedProvider string

	Applied vedrov1alpha1.BucketAppliedState
}
