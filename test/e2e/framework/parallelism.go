package framework

import (
	"errors"

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
