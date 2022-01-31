package framework

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) CreateNamespace() error {
	ns := &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.namespace,
		},
	}

	return f.kc.Create(f.ctx, ns)
}

func (f *Framework) DeleteNamespace() error {
	ns := &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.namespace,
		},
	}
	return f.kc.Delete(f.ctx, ns)
}
