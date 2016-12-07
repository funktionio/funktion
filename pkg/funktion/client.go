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

const resyncPeriod = 30 * time.Second

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

// CreateSubscriptionListOptions returns the default selector for Subscription resources
func CreateSubscriptionListOptions() (*api.ListOptions, error) {
	selector, err := labels.Parse("funktion.fabric8.io/kind=Subscription")
	if err != nil {
		return nil, err
	}
	listOpts := api.ListOptions{
		LabelSelector: selector,
	}
	return &listOpts, nil
}
// CreateConnectorListOptions returns the default selector for Connector resources
func CreateConnectorListOptions() (*api.ListOptions, error) {
	selector, err := labels.Parse("funktion.fabric8.io/kind=Connector")
	if err != nil {
		return nil, err
	}
	listOpts := api.ListOptions{
		LabelSelector: selector,
	}
	return &listOpts, nil
}