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
	"context"
	"time"

	"gomodules.xyz/pointer"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubedb.dev/apimachinery/apis/kubedb"
	kubedbv1 "kubedb.dev/apimachinery/apis/kubedb/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (f *Framework) getDatabaseNamespace() string {
	return f.namespace
}

func (f *Framework) newMongoDBStandaloneDatabase() *kubedbv1.MongoDB {
	return &kubedbv1.MongoDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("supervisor"),
			Namespace: f.getDatabaseNamespace(),
		},
		Spec: kubedbv1.MongoDBSpec{
			Version:     "4.2.3",
			StorageType: kubedbv1.StorageTypeDurable,
			Storage: &core.PersistentVolumeClaimSpec{
				AccessModes: []core.PersistentVolumeAccessMode{core.ReadWriteOnce},
				Resources: core.VolumeResourceRequirements{
					Requests: core.ResourceList{
						core.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				StorageClassName: pointer.StringP("standard"),
			},
			DeletionPolicy: "WipeOut",
		},
	}
}

func (f *Framework) newPostgresStandaloneDatabase(customAuthName string) *kubedbv1.Postgres {
	return &kubedbv1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("supervisor"),
			Namespace: f.getDatabaseNamespace(),
		},
		Spec: kubedbv1.PostgresSpec{
			Version:     "13.2",
			StorageType: kubedbv1.StorageTypeDurable,
			AuthSecret: &kubedbv1.SecretReference{
				LocalObjectReference: core.LocalObjectReference{
					Name: customAuthName,
				},
			},
			Storage: &core.PersistentVolumeClaimSpec{
				AccessModes: []core.PersistentVolumeAccessMode{core.ReadWriteOnce},
				Resources: core.VolumeResourceRequirements{
					Requests: core.ResourceList{
						core.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				StorageClassName: pointer.StringP("standard"),
			},
			DeletionPolicy: "WipeOut",
		},
	}
}

func (f *Framework) CreateNewStandaloneMongoDB() (*kubedbv1.MongoDB, error) {
	mongoDB := f.newMongoDBStandaloneDatabase()
	if err := f.kc.Create(f.ctx, mongoDB); err != nil {
		return nil, err
	}

	err := wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute*10, true, func(ctx context.Context) (bool, error) {
		mg := &kubedbv1.MongoDB{}
		key := client.ObjectKey{Namespace: mongoDB.Namespace, Name: mongoDB.Name}
		if err := f.kc.Get(f.ctx, key, mg); err != nil {
			return false, client.IgnoreNotFound(err)
		}

		if mg.Status.Phase == kubedb.DatabaseReady {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return mongoDB, nil
}

func (f *Framework) CreateNewStandalonePostgres() (*kubedbv1.Postgres, error) {
	pgAuth, err := f.createPostgresCustomAuthSecret()
	if err != nil {
		return nil, err
	}
	pg := f.newPostgresStandaloneDatabase(pgAuth.Name)
	if err := f.kc.Create(f.ctx, pg); err != nil {
		return nil, err
	}

	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute*10, true, func(ctx context.Context) (bool, error) {
		mg := &kubedbv1.Postgres{}
		key := client.ObjectKey{Namespace: pg.Namespace, Name: pg.Name}
		if err := f.kc.Get(f.ctx, key, mg); err != nil {
			return false, client.IgnoreNotFound(err)
		}

		if mg.Status.Phase == kubedb.DatabaseReady {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return pg, nil
}

func (f *Framework) DeleteMongoDB(key client.ObjectKey) error {
	mg := &kubedbv1.MongoDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}

	return f.kc.Delete(f.ctx, mg)
}

func (f *Framework) DeletePostgres(key client.ObjectKey) error {
	mg := &kubedbv1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}

	return f.kc.Delete(f.ctx, mg)
}

func (f *Framework) createPostgresCustomAuthSecret() (*core.Secret, error) {
	auth := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("supervisor-pg-auth-"),
			Namespace: f.postgresAuthNamespace(),
		},
		StringData: map[string]string{
			"username": "postgres",
			"password": "admin@1234",
		},
		Type: core.SecretTypeBasicAuth,
	}
	if err := f.kc.Create(f.ctx, auth); err != nil {
		return nil, err
	}

	err := wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		createdAuth := &core.Secret{}
		key := client.ObjectKey{Name: auth.Name, Namespace: auth.Namespace}
		if err := f.kc.Get(f.ctx, key, createdAuth); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return auth, nil
}

func (f *Framework) postgresAuthNamespace() string {
	return f.namespace
}
