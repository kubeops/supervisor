package shared

import (
	"k8s.io/apimachinery/pkg/runtime"
	"kubeops.dev/supervisor/apis"
	mongodb_ops "kubeops.dev/supervisor/pkg/mongodb-ops"
)

func GetExecutableObject(obj runtime.RawExtension) (apis.Executable, error) {
	// do it now only for MongoDB
	mgOps, err := mongodb_ops.NewMongoDBOpsRequest(obj)
	if err != nil {
		return nil, err
	}
	return mgOps, nil
}
