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

package k8sutil

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/conversion"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/tools/cache"
)

const (
	resyncPeriod = 30 * time.Second
)

type PodFunc func(pod *v1.Pod) error

type PodAction struct {
	OnPodChange PodFunc

	latestPodName string
	podInformer   cache.SharedIndexInformer
}

// V1BetaSelectorToListOptions converts a selector from a Deployment to an api.ListOptions object
func V1BetaSelectorToListOptions(selector *v1beta1.LabelSelector) (*api.ListOptions, error) {
	labelSelector := selector.MatchLabels
	if labelSelector == nil {
		return nil, fmt.Errorf("Selector does not havea matchLabels")
	}
	labelString := toLabelString(labelSelector)
	oldListOpts := v1beta1.ListOptions{
		LabelSelector: labelString,
	}
	newListOpts := api.ListOptions{}
	scope := conversion.Scope(nil)
	err := v1beta1.Convert_v1beta1_ListOptions_To_api_ListOptions(&oldListOpts, &newListOpts, scope)
	if err != nil {
		return nil, err
	}
	return &newListOpts, nil
}

func toLabelString(labels map[string]string) string {
	var buffer bytes.Buffer
	i := 0
	for k, v := range labels {
		i++
		if i > 1 {
			buffer.WriteString(",")
		}
		buffer.WriteString(fmt.Sprintf("%s=%s", k, v))
	}
	return buffer.String()
}

// WatchPods sets up the watcher for the given kubernetes client, namespace and listOpts
func (p *PodAction) WatchPods(kubeclient *kubernetes.Clientset, namespace string, listOpts *api.ListOptions) cache.SharedIndexInformer {
	resources := kubeclient.Pods(namespace)
	listWatch := cache.ListWatch{
		ListFunc: func(options api.ListOptions) (runtime.Object, error) {
			return resources.List(*listOpts)
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return resources.Watch(*listOpts)
		},
	}
	inf := cache.NewSharedIndexInformer(
		&listWatch,
		&v1.Pod{},
		resyncPeriod,
		cache.Indexers{},
	)

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    p.handlePodAdd,
		DeleteFunc: p.handlePodDelete,
		UpdateFunc: p.handlePodUpdate,
	})
	p.podInformer = inf
	return inf
}

func (p *PodAction) handlePodAdd(obj interface{}) {
	p.CheckLatestPod()
}

func (p *PodAction) handlePodUpdate(old, obj interface{}) {
	p.CheckLatestPod()
}

func (p *PodAction) handlePodDelete(obj interface{}) {
	p.CheckLatestPod()
}

// WatchLoop is the loop waiting or the watch to fail
func (p *PodAction) WatchLoop() error {
	stopc := make(chan struct{})
	errc := make(chan error)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		go p.podInformer.Run(stopc)
		<-stopc
		wg.Done()
	}()

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case <-term:
		fmt.Fprintln(os.Stderr)
		fmt.Println("Received SIGTERM, exiting gracefully...")
		close(stopc)
		wg.Wait()
	case err := <-errc:
		fmt.Printf("Unexpected error received: %v\n", err)
		close(stopc)
		wg.Wait()
		return err
	}
	return nil
}

func (p *PodAction) CheckLatestPod() {
	var latestPod *v1.Pod
	latestPodName := ""
	l := p.podInformer.GetStore().List()
	if l != nil {
		for _, obj := range l {
			if obj != nil {
				pod := obj.(*v1.Pod)
				if pod != nil {
					if isPodReady(pod) {
						if latestPod == nil || isPodNewer(pod, latestPod) {
							latestPod = pod
							latestPodName = pod.Name
						}
					}
				}
			}
		}
	}
	if latestPodName != p.latestPodName {
		p.latestPodName = latestPodName
		fn := p.OnPodChange
		if fn != nil {
			err := fn(latestPod)
			if err != nil {
				fmt.Printf("Unexpected error received: %v\n", err)
			}
		}
	}
}

func isPodReady(pod *v1.Pod) bool {
	status := pod.Status
	statusText := status.Phase
	if statusText == "Running" {
		for _, cond := range status.Conditions {
			if cond.Type == "Ready" {
				return cond.Status == "True"
			}
		}
	}
	return false
}

// isPodNewer returns true if a is newer than b
func isPodNewer(a *v1.Pod, b *v1.Pod) bool {
	t1 := a.ObjectMeta.CreationTimestamp
	t2 := b.ObjectMeta.CreationTimestamp
	return t2.Before(t1)
}
