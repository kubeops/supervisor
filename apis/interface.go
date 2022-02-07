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

package apis

import (
	"context"

	elaticsearch_ops "kubeops.dev/supervisor/pkg/elaticsearch-ops"
	mariadb_ops "kubeops.dev/supervisor/pkg/mariadb-ops"
	mongodb_ops "kubeops.dev/supervisor/pkg/mongodb-ops"
	mysql_ops "kubeops.dev/supervisor/pkg/mysql-ops"
	postgres_ops "kubeops.dev/supervisor/pkg/postgres-ops"
	redis_ops "kubeops.dev/supervisor/pkg/redis-ops"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OpsRequest interface {
	Execute(context.Context, client.Client) error
	WaitForOpsRequestToBeCompleted(context.Context, client.Client) error
}

var _ OpsRequest = &mongodb_ops.MongoDBOpsRequest{}
var _ OpsRequest = &elaticsearch_ops.ESOpsRequest{}
var _ OpsRequest = &postgres_ops.PostgresOpsRequest{}
var _ OpsRequest = &mysql_ops.MySQLOpsRequest{}
var _ OpsRequest = &mariadb_ops.MariaDBOpsRequest{}
var _ OpsRequest = &redis_ops.RedisOpsRequest{}
