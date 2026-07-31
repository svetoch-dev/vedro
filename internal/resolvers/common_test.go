package resolvers

import (
	vedroconditions "github.com/svetoch-dev/vedro/internal/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("isReady", func() {
	const generation int64 = 3

	It("reports a resource without conditions as not ready", func() {
		condition, ready := isReady(generation, nil)

		Expect(ready).To(BeFalse())
		Expect(condition).To(Equal(&metav1.Condition{
			Type:    vedroconditions.TypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  vedroconditions.ReasonNoConditions,
			Message: "No conditions found",
		}))
	})

	DescribeTable("evaluates existing conditions",
		func(conditions []metav1.Condition, expectedReady bool, expectedCondition *metav1.Condition) {
			condition, ready := isReady(generation, conditions)

			Expect(ready).To(Equal(expectedReady))
			Expect(condition).To(Equal(expectedCondition))
		},
		Entry("all conditions are true for the current generation",
			[]metav1.Condition{
				{
					Type:               vedroconditions.TypeReady,
					Status:             metav1.ConditionTrue,
					ObservedGeneration: generation,
				},
				{
					Type:               vedroconditions.TypeProviderConfigReady,
					Status:             metav1.ConditionTrue,
					ObservedGeneration: generation,
				},
			},
			true,
			nil,
		),
		Entry("a condition is false",
			[]metav1.Condition{
				{
					Type:               vedroconditions.TypeReady,
					Status:             metav1.ConditionFalse,
					ObservedGeneration: generation,
					Reason:             "ReconcileFailed",
					Message:            "reconciliation failed",
				},
			},
			false,
			&metav1.Condition{
				Type:               vedroconditions.TypeReady,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: generation,
				Reason:             "ReconcileFailed",
				Message:            "reconciliation failed",
			},
		),
		Entry("a condition is unknown",
			[]metav1.Condition{
				{
					Type:               vedroconditions.TypeReady,
					Status:             metav1.ConditionUnknown,
					ObservedGeneration: generation,
					Reason:             "Reconciling",
					Message:            "reconciliation is in progress",
				},
			},
			false,
			&metav1.Condition{
				Type:               vedroconditions.TypeReady,
				Status:             metav1.ConditionUnknown,
				ObservedGeneration: generation,
				Reason:             "Reconciling",
				Message:            "reconciliation is in progress",
			},
		),
		Entry("a true condition belongs to an older generation",
			[]metav1.Condition{
				{
					Type:               vedroconditions.TypeReady,
					Status:             metav1.ConditionTrue,
					ObservedGeneration: generation - 1,
				},
			},
			false,
			&metav1.Condition{
				Type:    vedroconditions.TypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  vedroconditions.ReasonGenerationMissmatch,
				Message: "Condition ObservedGeneration does not match CR generation",
			},
		),
	)
})
