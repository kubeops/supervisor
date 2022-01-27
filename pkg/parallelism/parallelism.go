package parallelism

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime/schema"
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
	} else if r.rcmd.Status.Parallelism == supervisorv1alpha1.QueuePerTarget {
		return r.isMaintainingQueuePerTarget()
	} else if r.rcmd.Status.Parallelism == supervisorv1alpha1.QueuePerTargetAndNamespace {
		return r.isMaintainingQueuePerTargetAndNamespace()
	}
	return false, errors.New("invalid parallelism type")
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

func (r *ParallelRunner) isMaintainingQueuePerTargetAndNamespace() (bool, error) {
	gv, err := schema.ParseGroupVersion(*r.rcmd.Spec.Target.APIGroup)
	if err != nil {
		return false, err
	}
	reqGK := schema.GroupKind{
		Group: gv.Group,
		Kind:  r.rcmd.Spec.Target.Kind,
	}

	rcmdList := &supervisorv1alpha1.RecommendationList{}
	if err := r.kc.List(r.ctx, rcmdList, client.InNamespace(r.rcmd.Namespace)); err != nil {
		return false, err
	}
	return isMaintainingQueuePerGK(reqGK, rcmdList)
}

func (r *ParallelRunner) isMaintainingQueuePerTarget() (bool, error) {
	gv, err := schema.ParseGroupVersion(*r.rcmd.Spec.Target.APIGroup)
	if err != nil {
		return false, err
	}
	reqGK := schema.GroupKind{
		Group: gv.Group,
		Kind:  r.rcmd.Spec.Target.Kind,
	}

	rcmdList := &supervisorv1alpha1.RecommendationList{}
	if err := r.kc.List(r.ctx, rcmdList); err != nil {
		return false, err
	}
	return isMaintainingQueuePerGK(reqGK, rcmdList)
}

func isMaintainingQueuePerGK(reqGK schema.GroupKind, rcList *supervisorv1alpha1.RecommendationList) (bool, error) {
	for _, rc := range rcList.Items {
		gv, err := schema.ParseGroupVersion(*rc.Spec.Target.APIGroup)
		if err != nil {
			return false, err
		}

		gk := schema.GroupKind{
			Group: gv.Group,
			Kind:  rc.Spec.Target.Kind,
		}
		if reqGK != gk {
			continue
		}

		if kmapi.IsConditionTrue(rc.Status.Conditions, supervisorv1alpha1.SuccessfullyCreatedOperation) &&
			!kmapi.HasCondition(rc.Status.Conditions, supervisorv1alpha1.SuccessfullyExecutedOperation) {
			return false, nil
		}
	}
	return true, nil
}
