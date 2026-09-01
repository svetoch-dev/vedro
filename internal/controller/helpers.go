package controller

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Reconciled() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func ReconcileError(ctx context.Context, err error, msg string, keysAndValues ...any) (reconcile.Result, error) {
	if err == nil {
		return Reconciled()
	}

	if msg == "" {
		msg = "reconcile failed"
	}

	log.FromContext(ctx).Error(err, msg, keysAndValues...)

	return reconcile.Result{}, err
}

func ReconcileAfter(
	ctx context.Context,
	duration time.Duration,
	msg string,
	keysAndValues ...any,
) (reconcile.Result, error) {
	if msg == "" {
		msg = fmt.Sprintf("reconcile failed. Requeuing after %v", duration)
	}

	log.FromContext(ctx).Info(msg, keysAndValues...)

	return reconcile.Result{
		RequeueAfter: duration,
	}, nil
}

func ReconcileIgnoreNotFound(ctx context.Context, err error, msg string, keysAndValues ...any) (reconcile.Result, error) {
	if err == nil || apierrors.IsNotFound(err) {
		return Reconciled()
	}

	return ReconcileError(ctx, err, msg, keysAndValues...)
}

func copyConditionState(dst *metav1.Condition, src metav1.Condition) {
	dst.Message = src.Message
	dst.Reason = src.Reason
	dst.Status = src.Status
}
