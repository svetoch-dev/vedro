package controller

import (
	"context"

	"github.com/svetoch-dev/vedro/internal/conditions"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/resolvers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type BucketSetupStatus int

const (
	BucketOk           BucketSetupStatus = 0
	BucketBeingDeleted BucketSetupStatus = 1
	BucketNotOk        BucketSetupStatus = 2
	BucketNotReady     BucketSetupStatus = 3
)

func prepareBucket(
	ctx context.Context,
	kubeClient client.Client,
	bucketRef vedro.BucketReference,
) (resolvers.BucketResolver, BucketSetupStatus) {
	logger := log.FromContext(ctx)

	bucket := resolvers.BucketResolver{
		KubeClient: kubeClient,
		Logger:     logger,
	}

	bucket.Resolve(ctx, types.NamespacedName{
		Namespace: bucketRef.Namespace,
		Name:      bucketRef.Name,
	})

	bucket.Condition.Type = conditions.TypeBucketReady

	if bucket.IsBeingDeleted() {
		logger.Info("deleting BucketAccess because referenced Bucket is terminating")
		return bucket, BucketBeingDeleted
	}

	if !bucket.IsOk() {
		return bucket, BucketNotOk
	}

	notReadyCondition, rdy := bucket.IsReady()

	if !rdy {
		logger.Info("Bucket is not Ready")
		copyConditionState(&bucket.Condition, *notReadyCondition)
		return bucket, BucketNotReady
	}

	bucket.Condition.Status = metav1.ConditionTrue
	bucket.Condition.Reason = conditions.ReasonBucketReady
	bucket.Condition.Message = "Bucket Ready"

	return bucket, BucketOk
}
