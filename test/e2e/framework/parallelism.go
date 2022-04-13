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

func (f *Framework) EnsureQueuePerTargetParallelism(stopCh chan bool, target metav1.GroupKind, ns string) error {
	for {
		select {
		case <-stopCh:
			return nil
		default:
			rcmdList := &api.RecommendationList{}
			if err := f.kc.List(f.ctx, rcmdList, client.InNamespace(ns)); err != nil {
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
				return errors.New("parallelism is not maintained")
			}
		}
	}
}
