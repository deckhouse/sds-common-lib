/*
Copyright 2025 Flant JSC

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

package kubeclient

import (
	"fmt"

	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func New(schemeFuncs ...func(*apiruntime.Scheme) error) (client.Client, error) {
	kubeConfig, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("getting config: %w", err)
	}

	scheme := apiruntime.NewScheme()

	for _, f := range schemeFuncs {
		if err = f(scheme); err != nil {
			return nil, fmt.Errorf("building scheme: %w", err)
		}
	}

	clientOpts := client.Options{
		Scheme: scheme,
	}

	return client.New(kubeConfig, clientOpts)
}
