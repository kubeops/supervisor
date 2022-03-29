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

package shared

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGVK(obj runtime.RawExtension) (schema.GroupVersionKind, error) {
	unObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(obj.Raw, unObj); err != nil {
		return schema.GroupVersionKind{}, err
	}
	return unObj.GetObjectKind().GroupVersionKind(), nil
}

func GetUnstructuredObj(obj runtime.RawExtension) (*unstructured.Unstructured, error) {
	unObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(obj.Raw, unObj); err != nil {
		return nil, err
	}
	return unObj, nil
}
