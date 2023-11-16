/*
Copyright 2023. projectsveltos.io. All rights reserved.

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

package utils

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Collector is the client that implements methods to collect resources and logs
type Collector struct {
	client        client.Client
	restConfig    *rest.Config
	clientset     *kubernetes.Clientset
	scheme        *runtime.Scheme
	configMapName string
	directory     string
}

var (
	collectorInstance *Collector
	mux               sync.Mutex
)

// GetCollectorInstance return k8sAccess instance used to access resources in the
// management cluster.
func GetCollectorInstance(scheme *runtime.Scheme, restConfig *rest.Config,
	directory, configMapName string) (*Collector, error) {

	mux.Lock()
	defer mux.Unlock()

	if collectorInstance == nil {
		cs, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			werr := fmt.Errorf("error in getting access to K8S: %w", err)
			return nil, werr
		}

		c, err := client.New(restConfig, client.Options{Scheme: scheme})
		if err != nil {
			werr := fmt.Errorf("failed to connect: %w", err)
			return nil, werr
		}

		collectorInstance = &Collector{
			scheme:        scheme,
			client:        c,
			clientset:     cs,
			restConfig:    restConfig,
			configMapName: configMapName,
			directory:     directory,
		}
	}

	return collectorInstance, nil
}

// GetScheme returns scheme
func (a *Collector) GetScheme() *runtime.Scheme {
	return a.scheme
}

// GetClient returns scheme
func (a *Collector) GetClient() client.Client {
	return a.client
}

// GetConfig returns restConfig
func (a *Collector) GetConfig() *rest.Config {
	return a.restConfig
}
