package resolvers

import (
	"context"

	cond "github.com/svetoch-dev/vedro/internal/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ResourceResolver interface {
	Resolve(ctx context.Context, name types.NamespacedName)
	IsOk() bool
	IsBeingDeleted() bool
	IsDeletingExternalResource() bool
	IsReady() (*metav1.Condition, bool)
}

func isReady(gen int64, conditions []metav1.Condition) (*metav1.Condition, bool) {
	if len(conditions) == 0 {
		return &metav1.Condition{
			Type:    cond.TypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  cond.ReasonNoConditions,
			Message: "No conditions found",
		}, false
	}
	for _, condition := range conditions {
		if condition.Status != metav1.ConditionTrue {
			return &condition, false
		}
		if gen != condition.ObservedGeneration {
			return &metav1.Condition{
				Type:    condition.Type,
				Status:  metav1.ConditionFalse,
				Reason:  cond.ReasonGenerationMissmatch,
				Message: "Condition ObservedGeneration does not match CR generation",
			}, false

		}
	}
	return nil, true
}
