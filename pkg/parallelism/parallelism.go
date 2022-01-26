package parallelism

import (
	"context"

	kmapi "kmodules.xyz/client-go/api/v1"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ParallelRunner struct {
	ctx  context.Context
	kc   client.Client
	rcmd *supervisorv1alpha1.Recommendation
}

func NewParallelRunner(ctx context.Context, kc client.Client, rcmd *supervisorv1alpha1.Recommendation) *ParallelRunner {
	return &ParallelRunner{
		ctx:  ctx,
		kc:   kc,
		rcmd: rcmd,
	}
}

func (r *ParallelRunner) MaintainParallelism() (bool, error) {
	if r.rcmd.Status.Parallelism == supervisorv1alpha1.QueuePerNamespace {
		return r.isMaintainingQueuePerNamespace()
	}
	return false, nil
}

func (r *ParallelRunner) isMaintainingQueuePerNamespace() (bool, error) {
	rcmdList := &supervisorv1alpha1.RecommendationList{}

	if err := r.kc.List(r.ctx, rcmdList, client.InNamespace(r.rcmd.Namespace)); err != nil {
		return false, err
	}

	for _, rc := range rcmdList.Items {
		if kmapi.IsConditionTrue(rc.Status.Conditions, supervisorv1alpha1.SuccessfullyCreatedOperation) &&
			!kmapi.HasCondition(rc.Status.Conditions, supervisorv1alpha1.SuccessfullyExecutedOperation) {
			return false, nil
		}
	}
	return true, nil
}
