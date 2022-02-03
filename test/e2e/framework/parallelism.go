package framework

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime/schema"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (f *Framework) EnsureQueuePerNamespaceParallelism(stopCh chan bool) error {
	for {
		select {
		case <-stopCh:
			return nil
		default:
			rcmdList := &api.RecommendationList{}
			if err := f.kc.List(f.ctx, rcmdList, client.InNamespace(f.namespace)); err != nil {
				return err
			}
			inProgressCnt := 0
			for _, r := range rcmdList.Items {
				if r.Status.Phase == api.InProgress {
					inProgressCnt++
				}
			}
			if inProgressCnt > 1 {
				return errors.New("QueuePerNamespace parallelism is not maintained")
			}
		}
	}
}

func (f *Framework) EnsureQueuePerTargetParallelism(stopCh chan bool, target metav1.GroupKind) error {
	for {
		select {
		case <-stopCh:
			return nil
		default:
			rcmdList := &api.RecommendationList{}
			if err := f.kc.List(f.ctx, rcmdList); err != nil {
				return err
			}

			inProgressCnt := 0

			for _, r := range rcmdList.Items {
				if r.Status.Phase == api.InProgress {
					gv, err := schema.ParseGroupVersion(*r.Spec.Target.APIGroup)
					if err != nil {
						return err
					}
					gk := metav1.GroupKind{
						Group: gv.Group,
						Kind:  r.Spec.Target.Kind,
					}
					if gk == target {
						inProgressCnt += 1
					}
				}
			}
			if inProgressCnt > 1 {
				return errors.New("QueuePerTarget is not maintained")
			}
		}
	}
}
