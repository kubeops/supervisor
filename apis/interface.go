package apis

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Executable interface {
	Execute(client.Client) error
}
