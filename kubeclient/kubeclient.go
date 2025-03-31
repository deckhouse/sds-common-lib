package kubeclient

import (
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func New(kubeconfigPath string, schemeFuncs ...func(*apiruntime.Scheme) error) (client.Client, error) {
	kubeConfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	scheme := apiruntime.NewScheme()

	for _, f := range schemeFuncs {
		if err = f(scheme); err != nil {
			return nil, err
		}
	}

	clientOpts := client.Options{
		Scheme: scheme,
	}

	return client.New(kubeConfig, clientOpts)
}
