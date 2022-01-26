package parallelism

import (
	"context"

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
	return false, nil
}
