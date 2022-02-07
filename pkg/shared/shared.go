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
	"errors"
	"fmt"
	"os"

	"kubeops.dev/supervisor/apis"
	elaticsearch_ops "kubeops.dev/supervisor/pkg/elaticsearch-ops"
	mariadb_ops "kubeops.dev/supervisor/pkg/mariadb-ops"
	mongodb_ops "kubeops.dev/supervisor/pkg/mongodb-ops"
	mysql_ops "kubeops.dev/supervisor/pkg/mysql-ops"
	postgres_ops "kubeops.dev/supervisor/pkg/postgres-ops"
	redis_ops "kubeops.dev/supervisor/pkg/redis-ops"

	"github.com/jonboulle/clockwork"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
)

func GetOpsRequestObject(obj runtime.RawExtension) (apis.OpsRequest, error) {
	gvk, err := GetGVK(obj)
	if err != nil {
		return nil, err
	}

	if gvk.Group == opsapi.SchemeGroupVersion.Group && gvk.Kind == opsapi.ResourceKindMongoDBOpsRequest {
		return mongodb_ops.NewMongoDBOpsRequest(obj)
	}

	if gvk.Group == opsapi.SchemeGroupVersion.Group && gvk.Kind == opsapi.ResourceKindElasticsearchOpsRequest {
		return elaticsearch_ops.NewESOpsRequest(obj)
	}

	if gvk.Group == opsapi.SchemeGroupVersion.Group && gvk.Kind == opsapi.ResourceKindPostgresOpsRequest {
		return postgres_ops.NewPostgresOpsRequest(obj)
	}

	if gvk.Group == opsapi.SchemeGroupVersion.Group && gvk.Kind == opsapi.ResourceKindMySQLOpsRequest {
		return mysql_ops.NewMySQLOpsRequest(obj)
	}

	if gvk.Group == opsapi.SchemeGroupVersion.Group && gvk.Kind == opsapi.ResourceKindMariaDBOpsRequest {
		return mariadb_ops.NewMariaDBOpsRequest(obj)
	}

	if gvk.Group == opsapi.SchemeGroupVersion.Group && gvk.Kind == opsapi.ResourceKindRedisOpsRequest {
		return redis_ops.NewRedisOpsRequest(obj)
	}

	return nil, fmt.Errorf("invalid operation, Group: %v Kind: %v is not supported", gvk.Group, gvk.Kind)
}

func GetGVK(obj runtime.RawExtension) (schema.GroupVersionKind, error) {
	unObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(obj.Raw, unObj); err != nil {
		return schema.GroupVersionKind{}, err
	}
	return unObj.GetObjectKind().GroupVersionKind(), nil
}

func GetType(obj runtime.RawExtension) (string, error) {
	unObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(obj.Raw, unObj); err != nil {
		return "", err
	}
	spec, ok := unObj.Object["spec"].(map[string]interface{})
	if !ok {
		return "", errors.New("failed to parse spec section of raw operation object")
	}
	opsType, ok := spec["type"].(string)
	if !ok {
		return "", errors.New("failed to parse .spec.type from raw operation object")
	}
	return opsType, nil
}

func GetClock() clockwork.Clock {
	if os.Getenv("APPSCODE_SUPERVISOR_TEST") == "TRUE" {
		return clockwork.NewFakeClock()
	}
	return clockwork.NewRealClock()
}
