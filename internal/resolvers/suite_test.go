package resolvers

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestResolvers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resolvers Suite")
}
