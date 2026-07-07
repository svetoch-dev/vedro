package yc

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("isNotFound", func() {
	DescribeTable("classifies YC API errors",
		func(err error, want bool) {
			Expect(isNotFound(err)).To(Equal(want))
		},
		Entry(
			"not found status",
			status.Error(codes.NotFound, "bucket not found"),
			true,
		),
		Entry(
			"wrapped not found status",
			fmt.Errorf("delete bucket: %w", status.Error(codes.NotFound, "bucket not found")),
			true,
		),
		Entry(
			"permission denied status",
			status.Error(codes.PermissionDenied, "permission denied"),
			false,
		),
		Entry(
			"plain error",
			fmt.Errorf("network error"),
			false,
		),
	)
})
