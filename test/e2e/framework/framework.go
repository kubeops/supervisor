package framework

import (
	"context"
	"os"

	"github.com/jonboulle/clockwork"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	"gomodules.xyz/x/crypto/rand"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Framework struct {
	ctx           context.Context
	restConfig    *rest.Config
	kc            client.Client
	namespace     string
	name          string
	clusterMWName string
	clock         clockwork.Clock
}

type Invocation struct {
	*Framework
	app string
}

func New(ctx context.Context, restConfig *rest.Config, kc client.Client) *Framework {
	return &Framework{
		ctx:           ctx,
		restConfig:    restConfig,
		kc:            kc,
		namespace:     rand.WithUniqSuffix("supervisor-test-ns"),
		name:          rand.WithUniqSuffix("supervisor"),
		clusterMWName: rand.WithUniqSuffix("cluster-mw"),
		clock:         api.GetClock(),
	}
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework: f,
		app:       rand.WithUniqSuffix("supervisor-e2e"),
	}
}

func (i *Invocation) RestConfig() *rest.Config {
	return i.restConfig
}

func (f *Framework) Name() string {
	return f.name
}

func (f *Framework) Namespace() string {
	return f.namespace
}

func (f *Framework) SetTestEnv() error {
	return os.Setenv(api.TestEnvKey, api.TestEnvVal)
}
