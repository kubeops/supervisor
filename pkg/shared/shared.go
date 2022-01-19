package shared

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"kubeops.dev/supervisor/apis"
	mongodb_ops "kubeops.dev/supervisor/pkg/mongodb-ops"
)

func GetOpsRequestObject(obj runtime.RawExtension) (apis.OpsRequest, error) {
	unObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(obj.Raw, unObj); err != nil {
		return nil, err
	}
	gvk := unObj.GetObjectKind().GroupVersionKind()

	if gvk.Group == opsapi.SchemeGroupVersion.Group && gvk.Kind == opsapi.ResourceKindMongoDBOpsRequest {
		return mongodb_ops.NewMongoDBOpsRequest(obj)
	}
	return nil, fmt.Errorf("invalid operation, Group: %v Kind: %v is not supported", gvk.Group, gvk.Kind)
}
