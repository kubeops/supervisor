package deadline_manager

import (
	"time"

	"github.com/jonboulle/clockwork"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/shared"
)

type manager struct {
	rcmd  *supervisorv1alpha1.Recommendation
	clock clockwork.Clock
}

func NewManager(rcmd *supervisorv1alpha1.Recommendation) *manager {
	return &manager{
		rcmd:  rcmd,
		clock: shared.GetClock(),
	}
}

func (m *manager) IsDeadlineLessThanADay() bool {
	return m.IsDeadlineLessThan(time.Hour * 24)
}

func (m *manager) IsDeadlineLessThan(duration time.Duration) bool {
	if m.rcmd.Spec.Deadline == nil {
		return false
	}
	now := m.clock.Now().UTC().Unix()
	deadline := m.rcmd.Spec.Deadline.UTC().Unix()
	dur := int64(duration.Seconds())

	return deadline-now < dur
}
