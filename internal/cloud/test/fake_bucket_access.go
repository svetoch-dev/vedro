package cloudtest

import (
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewBucketAccessCR(
	name string,
	mods ...func(*vedro.BucketAccess),
) vedro.BucketAccess {
	ba := vedro.BucketAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: vedro.BucketAccessSpec{
			BucketRef: vedro.BucketReference{
				Name:      "test-bucket",
				Namespace: "default",
			},
			PrincipalRef: vedro.PrincipalReference{
				Name:      "test-principal",
				Namespace: "default",
			},
			Access: vedro.Access{
				Level: vedro.ObjectReader,
			},
		},
	}
	for _, m := range mods {
		m(&ba)
	}
	return ba
}
