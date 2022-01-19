package apis

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OpsRequest interface {
	Execute(client.Client) error
}
