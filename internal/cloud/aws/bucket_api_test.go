package aws

import (
	"fmt"

	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("isS3NotFoundLike", func() {
	DescribeTable("classifies S3 API errors",
		func(err error, codes []string, want bool) {
			Expect(isS3NotFoundLike(err, codes...)).To(Equal(want))
		},
		Entry(
			"matching generic api error",
			&smithy.GenericAPIError{
				Code:    "NoSuchBucket",
				Message: "bucket does not exist",
			},
			[]string{"NoSuchBucket", "NotFound"},
			true,
		),
		Entry(
			"matching wrapped api error",
			fmt.Errorf("list versions: %w", &smithy.OperationError{
				ServiceID:     "S3",
				OperationName: "ListObjectVersions",
				Err: &smithy.GenericAPIError{
					Code:    "NotFound",
					Message: "not found",
				},
			}),
			[]string{"NoSuchBucket", "NotFound"},
			true,
		),
		Entry(
			"different api error",
			&smithy.GenericAPIError{
				Code:    "AccessDenied",
				Message: "access denied",
			},
			[]string{"NoSuchBucket", "NotFound"},
			false,
		),
		Entry(
			"plain error",
			fmt.Errorf("network error"),
			[]string{"NoSuchBucket", "NotFound"},
			false,
		),
	)
})
