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
