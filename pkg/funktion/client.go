//  Copyright 2016 Red Hat, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package funktion

import (
	"time"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/tools/cache"
)

const (
	// KindLabel is the label key used on ConfigMaps to indicate the kind of resource
	KindLabel = "funktion.fabric8.io/kind"

	// Subscription

	// ConnectorLabel is the label key used on a ConfigMap to refer to a Connector
	ConnectorLabel = "connector"

	// Function

	// RuntimeLabel is the label key used on a ConfigMap to refer to a Runtime
	RuntimeLabel = "runtime"

	// ConnectorKind is the value of a Connector fo the KindLabel
	ConnectorKind = "Connector"
	// SubscriptionKind is the value of a Subscription fo the KindLabel
	SubscriptionKind = "Subscription"
	// RuntimeKind is the value of a Runtime fo the KindLabel
	RuntimeKind = "Runtime"
	// FunctionKind is the value of a Function fo the KindLabel
	FunctionKind = "Function"
	// DeploymentKind is the value of a Deployment fo the KindLabel
	DeploymentKind = "Deployment"
	// ServiceKind is the value of a ConneServicector fo the KindLabel
	ServiceKind = "Service"

	resyncPeriod = 30 * time.Second
)

// NewConfigMapListWatch returns a new ListWatch for ConfigMaps with the given listOptions
func NewConfigMapListWatch(client *kubernetes.Clientset, listOpts api.ListOptions) *cache.ListWatch {
	configMaps := client.ConfigMaps(api.NamespaceAll)

	return &cache.ListWatch{
		ListFunc: func(options api.ListOptions) (runtime.Object, error) {
			return configMaps.List(listOpts)
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return configMaps.Watch(listOpts)
		},
	}
}

// NewServiceListWatch creates a watch on services
func NewServiceListWatch(client *kubernetes.Clientset) *cache.ListWatch {
	listOpts := api.ListOptions{}
	services := client.Services(api.NamespaceAll)
	return &cache.ListWatch{
		ListFunc: func(options api.ListOptions) (runtime.Object, error) {
			return services.List(listOpts)
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return services.Watch(listOpts)
		},
	}
}

// CreateSubscriptionListOptions returns the default selector for Subscription resources
func CreateSubscriptionListOptions() (*api.ListOptions, error) {
	return createKindListOptions(SubscriptionKind)
}

// CreateConnectorListOptions returns the default selector for Connector resources
func CreateConnectorListOptions() (*api.ListOptions, error) {
	return createKindListOptions(ConnectorKind)
}

// CreateRuntimeListOptions returns the default selector for Runtime resources
func CreateRuntimeListOptions() (*api.ListOptions, error) {
	return createKindListOptions(RuntimeKind)
}

// CreateFunctionListOptions returns the default selector for Function resources
func CreateFunctionListOptions() (*api.ListOptions, error) {
	return createKindListOptions(FunctionKind)
}

// CreateKindListOptions returns the selector for a given kind of resources
func createKindListOptions(kind string) (*api.ListOptions, error) {
	selector, err := labels.Parse(KindLabel + "=" + kind)
	if err != nil {
		return nil, err
	}
	listOpts := api.ListOptions{
		LabelSelector: selector,
	}
	return &listOpts, nil
}
