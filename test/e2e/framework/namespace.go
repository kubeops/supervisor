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
