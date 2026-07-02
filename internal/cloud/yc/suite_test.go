package yc

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestYC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "YC Suite")
}
