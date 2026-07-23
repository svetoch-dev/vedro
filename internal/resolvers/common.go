package resolvers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ResourceResolver interface {
	Resolve(ctx context.Context, name types.NamespacedName)
	IsOk() bool
	IsReady() (*metav1.Condition, bool)
}

func isReady(conditions []metav1.Condition) (*metav1.Condition, bool) {
	for _, condition := range conditions {
		if condition.Status == metav1.ConditionFalse {
			return &condition, false
		}
	}
	return nil, true
}
