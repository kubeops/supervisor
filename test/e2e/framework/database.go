package framework

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"gomodules.xyz/pointer"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubedbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
)

func (f *Framework) getDatabaseNamespace() string {
	return f.namespace
}

func (f *Framework) newMongoDBStandaloneDatabase() *kubedbapi.MongoDB {
	return &kubedbapi.MongoDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("supervisor"),
			Namespace: f.getDatabaseNamespace(),
		},
		Spec: kubedbapi.MongoDBSpec{
			Version:     "4.2.3",
			StorageType: kubedbapi.StorageTypeDurable,
			Storage: &core.PersistentVolumeClaimSpec{
				AccessModes: []core.PersistentVolumeAccessMode{core.ReadWriteOnce},
				Resources: core.ResourceRequirements{
					Requests: core.ResourceList{
						core.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				StorageClassName: pointer.StringP("standard"),
			},
			TerminationPolicy: "WipeOut",
		},
	}
}

func (f *Framework) CreateNewStandaloneMongoDB() (*kubedbapi.MongoDB, error) {
	mongoDB := f.newMongoDBStandaloneDatabase()
	if err := f.kc.Create(f.ctx, mongoDB); err != nil {
		return nil, err
	}

	err := wait.PollImmediate(time.Second, time.Minute*10, func() (bool, error) {
		mg := &kubedbapi.MongoDB{}
		key := client.ObjectKey{Namespace: mongoDB.Namespace, Name: mongoDB.Name}
		if err := f.kc.Get(f.ctx, key, mg); err != nil {
			return false, client.IgnoreNotFound(err)
		}

		if mg.Status.Phase == kubedbapi.DatabaseReady {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return mongoDB, nil
}

func (f *Framework) DeleteMongoDB(key client.ObjectKey) error {
	mg := &kubedbapi.MongoDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}

	return f.kc.Delete(f.ctx, mg)
}
