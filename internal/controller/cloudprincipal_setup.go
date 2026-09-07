package controller

import (
	"context"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/resolvers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CloudPrincipalSetupStatus int

const (
	CloudPrincipalOk           CloudPrincipalSetupStatus = 0
	CloudPrincipalBeingDeleted CloudPrincipalSetupStatus = 1
	CloudPrincipalNotOk        CloudPrincipalSetupStatus = 2
	CloudPrincipalNotReady     CloudPrincipalSetupStatus = 3
)

func prepareCloudPrincipal(
	ctx context.Context,
	kubeClient client.Client,
	principalRef vedro.PrincipalReference,
) (resolvers.CloudPrincipalResolver, CloudPrincipalSetupStatus) {
	logger := log.FromContext(ctx)

	principal := resolvers.CloudPrincipalResolver{
		KubeClient: kubeClient,
		Logger:     logger,
	}

	principal.Resolve(ctx, types.NamespacedName{
		Namespace: principalRef.Namespace,
		Name:      principalRef.Name,
	})

	principal.Condition.Type = conditions.TypeCloudPrincipalReady

	if principal.IsBeingDeleted() {
		logger.Info("deleting BucketAccess because referenced CloudPrincipal is terminating")
		return principal, CloudPrincipalBeingDeleted
	}

	if !principal.IsOk() {
		return principal, CloudPrincipalNotOk
	}

	notReadyCondition, rdy := principal.IsReady()

	if !rdy {
		logger.Info("Principal is not Ready")
		copyConditionState(&principal.Condition, *notReadyCondition)
		return principal, CloudPrincipalNotReady
	}

	principal.Condition.Status = metav1.ConditionTrue
	principal.Condition.Reason = conditions.ReasonCloudPrincipalReady
	principal.Condition.Message = "CloudPrincipal Ready"

	return principal, CloudPrincipalOk

}
