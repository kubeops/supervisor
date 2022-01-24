package policy

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/shared"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApprovalPolicyFinder struct {
	ctx  context.Context
	kc   client.Client
	rcmd *supervisorv1alpha1.Recommendation
}

func NewApprovalPolicyFinder(ctx context.Context, kc client.Client, rcmd *supervisorv1alpha1.Recommendation) *ApprovalPolicyFinder {
	return &ApprovalPolicyFinder{
		ctx:  ctx,
		kc:   kc,
		rcmd: rcmd,
	}
}

func (c *ApprovalPolicyFinder) FindApprovalPolicy() (*supervisorv1alpha1.ApprovalPolicy, error) {
	policyList := &supervisorv1alpha1.ApprovalPolicyList{}
	if err := c.kc.List(c.ctx, policyList, client.InNamespace(c.rcmd.Namespace)); err != nil {
		return nil, err
	}
	gv, err := schema.ParseGroupVersion(*c.rcmd.Spec.Target.APIGroup)
	if err != nil {
		return nil, err
	}
	opsGVK, err := shared.GetGVK(c.rcmd.Spec.Operation)
	if err != nil {
		return nil, err
	}
	targetOpsGK := metav1.GroupKind{
		Group: opsGVK.Group,
		Kind:  opsGVK.Kind,
	}

	opsType, err := shared.GetType(c.rcmd.Spec.Operation)
	if err != nil {
		return nil, err
	}
	targetObjGk := metav1.GroupKind{
		Group: gv.Group,
		Kind:  c.rcmd.Spec.Target.Kind,
	}

	for _, p := range policyList.Items {
		for _, t := range p.Targets {
			if isMatched(t, targetObjGk, targetOpsGK, opsType) {
				return &p, nil
			}
		}
	}
	return nil, nil
}

func isMatched(ref supervisorv1alpha1.TargetRef, targetObjGK, targetOpsGK metav1.GroupKind, opsType string) bool {
	if ref.Group == targetObjGK.Group && ref.Kind == targetObjGK.Kind {
		for _, op := range ref.Operations {
			if op.GroupKind == targetOpsGK && op.Type == opsType {
				return true
			}
		}
	}
	return false
}
