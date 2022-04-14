/*
Copyright AppsCode Inc. and Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package policy

import (
	"context"
	"errors"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/shared"

	"gomodules.xyz/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApprovalPolicyFinder struct {
	ctx  context.Context
	kc   client.Client
	rcmd *api.Recommendation
}

func NewApprovalPolicyFinder(ctx context.Context, kc client.Client, rcmd *api.Recommendation) *ApprovalPolicyFinder {
	return &ApprovalPolicyFinder{
		ctx:  ctx,
		kc:   kc,
		rcmd: rcmd,
	}
}

func (c *ApprovalPolicyFinder) FindApprovalPolicy() (*api.ApprovalPolicy, error) {
	policyList := &api.ApprovalPolicyList{}
	if err := c.kc.List(c.ctx, policyList, client.InNamespace(c.rcmd.Namespace)); err != nil {
		return nil, err
	}
	if c.rcmd.Spec.Target.APIGroup == nil {
		return nil, errors.New("target APIGroup is not provided")
	}
	opsGVK, err := shared.GetGVK(c.rcmd.Spec.Operation)
	if err != nil {
		return nil, err
	}
	targetOpsGK := metav1.GroupKind{
		Group: opsGVK.Group,
		Kind:  opsGVK.Kind,
	}

	targetObjGk := metav1.GroupKind{
		Group: pointer.String(c.rcmd.Spec.Target.APIGroup),
		Kind:  c.rcmd.Spec.Target.Kind,
	}

	for _, p := range policyList.Items {
		for _, t := range p.Targets {
			if isMatched(t, targetObjGk, targetOpsGK) {
				return &p, nil
			}
		}
	}
	return nil, nil
}

func isMatched(ref api.TargetRef, targetObjGK, targetOpsGK metav1.GroupKind) bool {
	if ref.Group == targetObjGK.Group && ref.Kind == targetObjGK.Kind {
		for _, op := range ref.Operations {
			if op.GroupKind == targetOpsGK {
				return true
			}
		}
	}
	return false
}
