/*
Copyright 2024.

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

package globals

import (

	//

	//
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewKubernetesClient return a new Kubernetes Dynamic client from client-go SDK
func NewKubernetesClient() (client *dynamic.DynamicClient, coreClient *kubernetes.Clientset, err error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return client, coreClient, err
	}

	// Create the clients to do requests to our friend: Kubernetes
	// Dynamic client
	client, err = dynamic.NewForConfig(config)
	if err != nil {
		return client, coreClient, err
	}

	// Core client
	coreClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return client, coreClient, err
	}

	return client, coreClient, err
}
