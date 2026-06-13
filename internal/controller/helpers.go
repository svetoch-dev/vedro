package controller

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

func ReconcileIgnoreNotFound(ctx context.Context, err error, msg string, keysAndValues ...any) (reconcile.Result, error) {
	if err == nil || apierrors.IsNotFound(err) {
		return Reconciled()
	}

	return ReconcileError(ctx, err, msg, keysAndValues...)
}
