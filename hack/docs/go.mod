module github.com/svetoch-dev/vedro/hack/docs

go 1.27.3

require (
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.1-0.20241111191808-71fefeed8910 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	// The generator's original x/tools dependency does not support Go 1.27.
	golang.org/x/tools v0.52.0 // indirect
	k8s.io/gengo/v2 v2.0.0-20240404160639-a0386bf69313 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
)

tool github.com/ahmetb/gen-crd-api-reference-docs
