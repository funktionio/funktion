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
	"fmt"
	"time"

	"github.com/funktionio/funktion/pkg/analytics"
	"github.com/funktionio/funktion/pkg/queue"

	"strings"

	"github.com/go-kit/kit/log"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	utilruntime "k8s.io/client-go/1.5/pkg/util/runtime"
	"k8s.io/client-go/1.5/pkg/util/wait"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
)

// Operator manages Funktion Deployments
type Operator struct {
	kclient *kubernetes.Clientset
	//funktionClient *rest.RESTClient
	logger log.Logger

	connectorInf    cache.SharedIndexInformer
	flowInf cache.SharedIndexInformer
	runtimeInf      cache.SharedIndexInformer
	functionInf     cache.SharedIndexInformer
	deploymentInf   cache.SharedIndexInformer
	serviceInf      cache.SharedIndexInformer

	queue *queue.Queue
}

// ResourceKey represents a kind and a key
type ResourceKey struct {
	Kind string
	Key  string
}

// New creates a new controller.
func New(cfg *rest.Config, logger log.Logger) (*Operator, error) {
	logger.Log("msg", "starting up!")
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := &Operator{
		kclient: client,
		logger:  logger,
		queue:   queue.New(),
	}

	logger.Log("msg", "creating ListOptions")
	flowListOpts, err := CreateFlowListOptions()
	if err != nil {
		return nil, err
	}
	connectorListOpts, err := CreateConnectorListOptions()
	if err != nil {
		return nil, err
	}
	runtimeListOpts, err := CreateRuntimeListOptions()
	if err != nil {
		return nil, err
	}
	functionListOpts, err := CreateFunctionListOptions()
	if err != nil {
		return nil, err
	}

	c.connectorInf = cache.NewSharedIndexInformer(
		NewConfigMapListWatch(c.kclient, *connectorListOpts),
		&v1.ConfigMap{},
		resyncPeriod,
		cache.Indexers{},
	)
	c.flowInf = cache.NewSharedIndexInformer(
		NewConfigMapListWatch(c.kclient, *flowListOpts),
		&v1.ConfigMap{},
		resyncPeriod,
		cache.Indexers{},
	)
	c.runtimeInf = cache.NewSharedIndexInformer(
		NewConfigMapListWatch(c.kclient, *runtimeListOpts),
		&v1.ConfigMap{},
		resyncPeriod,
		cache.Indexers{},
	)
	c.functionInf = cache.NewSharedIndexInformer(
		NewConfigMapListWatch(c.kclient, *functionListOpts),
		&v1.ConfigMap{},
		resyncPeriod,
		cache.Indexers{},
	)
	c.deploymentInf = cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(c.kclient.Extensions().GetRESTClient(), "deployments", api.NamespaceAll, nil),
		&v1beta1.Deployment{},
		resyncPeriod,
		cache.Indexers{},
	)
	c.serviceInf = cache.NewSharedIndexInformer(
		NewServiceListWatch(c.kclient),
		&v1.Service{},
		resyncPeriod,
		cache.Indexers{},
	)

	c.connectorInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddConnector,
		DeleteFunc: c.handleDeleteConnector,
		UpdateFunc: c.handleUpdateConnector,
	})
	c.flowInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddFlow,
		DeleteFunc: c.handleDeleteFlow,
		UpdateFunc: c.handleUpdateFlow,
	})
	c.runtimeInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddRuntime,
		DeleteFunc: c.handleDeleteRuntime,
		UpdateFunc: c.handleUpdateRuntime,
	})
	c.functionInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddFunction,
		DeleteFunc: c.handleDeleteFunction,
		UpdateFunc: c.handleUpdateFunction,
	})
	c.deploymentInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// TODO only look at Funktion related Deployments?
		AddFunc: func(d interface{}) {
			c.handleAddDeployment(d)
		},
		DeleteFunc: func(d interface{}) {
			c.handleDeleteDeployment(d)
		},
		UpdateFunc: func(old, cur interface{}) {
			c.handleUpdateDeployment(old, cur)
		},
	})
	c.serviceInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// TODO only look at Funktion related Services?
		AddFunc: func(d interface{}) {
			c.handleAddService(d)
		},
		DeleteFunc: func(d interface{}) {
			c.handleDeleteService(d)
		},
		UpdateFunc: func(old, cur interface{}) {
			c.handleUpdateService(old, cur)
		},
	})

	logger.Log("msg", "started up!")

	return c, nil
}

// Run the controller.
func (c *Operator) Run(stopc <-chan struct{}) error {
	defer c.queue.ShutDown()

	go c.worker()

	go c.connectorInf.Run(stopc)
	go c.flowInf.Run(stopc)
	go c.runtimeInf.Run(stopc)
	go c.functionInf.Run(stopc)
	go c.deploymentInf.Run(stopc)
	go c.serviceInf.Run(stopc)

	<-stopc
	return nil
}

func (c *Operator) keyFunc(obj interface{}) (string, bool) {
	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Log("msg", "creating key failed", "err", err)
		return k, false
	}
	return k, true
}

func (c *Operator) handleAddRuntime(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.RuntimeCreated()
	c.logger.Log("msg", "Runtime added", "key", key)
	c.enqueue(key, RuntimeKind)
}

func (c *Operator) handleDeleteRuntime(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.RuntimeDeleted()
	c.logger.Log("msg", "Runtime deleted", "key", key)
	c.enqueue(key, RuntimeKind)
}

func (c *Operator) handleUpdateRuntime(old, cur interface{}) {
	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	c.logger.Log("msg", "Runtime updated", "key", key)
	c.enqueue(key, RuntimeKind)
}

func (c *Operator) handleAddFunction(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.FunctionCreated()
	c.logger.Log("msg", "Function added", "key", key)
	c.enqueue(key, FunctionKind)
}

func (c *Operator) handleDeleteFunction(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.FunctionDeleted()
	c.logger.Log("msg", "Function deleted", "key", key)
	c.enqueue(key, FunctionKind)
}

func (c *Operator) handleUpdateFunction(old, cur interface{}) {
	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	c.logger.Log("msg", "Function updated", "key", key)
	c.enqueue(key, FunctionKind)
}

func (c *Operator) handleAddConnector(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.ConnectorCreated()
	c.logger.Log("msg", "Connector added", "key", key)
	c.enqueue(key, ConnectorKind)
}

func (c *Operator) handleDeleteConnector(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.ConnectorDeleted()
	c.logger.Log("msg", "Connector deleted", "key", key)
	c.enqueue(key, ConnectorKind)
}

func (c *Operator) handleUpdateConnector(old, cur interface{}) {
	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	c.logger.Log("msg", "Connector updated", "key", key)
	c.enqueue(key, ConnectorKind)
}

func (c *Operator) handleAddFlow(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.FlowCreated()
	c.logger.Log("msg", "Flow added", "key", key)
	c.enqueue(key, FlowKind)
}

func (c *Operator) handleDeleteFlow(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.FlowDeleted()
	c.logger.Log("msg", "Flow deleted", "key", key)
	c.enqueue(key, FlowKind)
}

func (c *Operator) handleUpdateFlow(old, cur interface{}) {
	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	c.logger.Log("msg", "Flow updated", "key", key)
	c.enqueue(key, FlowKind)
}

func (c *Operator) handleDeleteDeployment(obj interface{}) {
	if d := c.flowForDeployment(obj); d != nil {
		c.enqueue(d, DeploymentKind)
	}
}

func (c *Operator) handleAddDeployment(obj interface{}) {
	if d := c.flowForDeployment(obj); d != nil {
		c.enqueue(d, DeploymentKind)
	}
}

func (c *Operator) handleUpdateDeployment(oldo, curo interface{}) {
	old := oldo.(*v1beta1.Deployment)
	cur := curo.(*v1beta1.Deployment)

	//c.logger.Log("msg", "update handler", "old", old.ResourceVersion, "cur", cur.ResourceVersion, "kind", "Service"))

	// Periodic resync may resend the deployment without changes in-between.
	// Also breaks loops created by updating the resource ourselves.
	if old.ResourceVersion == cur.ResourceVersion {
		return
	}

	// Wake up Flow resource the deployment belongs to.
	if k := c.flowForDeployment(cur); k != nil {
		c.enqueue(k, DeploymentKind)
	}
}
func (c *Operator) handleDeleteService(obj interface{}) {
	if d := c.functionForService(obj); d != nil {
		c.enqueue(d, ServiceKind)
	}
}

func (c *Operator) handleAddService(obj interface{}) {
	if d := c.functionForService(obj); d != nil {
		c.enqueue(d, ServiceKind)
	}
}

func (c *Operator) handleUpdateService(oldo, curo interface{}) {
	old := oldo.(*v1.Service)
	cur := curo.(*v1.Service)

	//c.logger.Log("msg", "update handler", "old", old.ResourceVersion, "cur", cur.ResourceVersion, "kind", "Service")

	// Periodic resync may resend the service without changes in-between.
	// Also breaks loops created by updating the resource ourselves.
	if old.ResourceVersion == cur.ResourceVersion {
		return
	}

	// Wake up Funktion resource the service belongs to.
	if k := c.functionForService(cur); k != nil {
		c.enqueue(k, ServiceKind)
	}
}

// enqueue adds a key to the queue. If obj is a key already it gets added directly.
// Otherwise, the key is extracted via keyFunc.
func (c *Operator) enqueue(obj interface{}, kind string) {
	if obj == nil {
		return
	}

	key, ok := obj.(string)
	if !ok {
		key, ok = c.keyFunc(obj)
		if !ok {
			return
		}
	}

	c.queue.Add(&ResourceKey{
		Key:  key,
		Kind: kind,
	})
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (c *Operator) worker() {
	for {
		key, quit := c.queue.Get()
		if quit {
			return
		}
		if err := c.sync(key.(*ResourceKey)); err != nil {
			utilruntime.HandleError(fmt.Errorf("reconciliation failed, re-enqueueing: %s", err))
			// We only mark the item as done after waiting. In the meantime
			// other items can be processed but the same item won't be processed again.
			// This is a trivial form of rate-limiting that is sufficient for our throughput
			// and latency expectations.
			go func() {
				time.Sleep(3 * time.Second)
				c.queue.Done(key)
			}()
			continue
		}

		c.queue.Done(key)
	}
}

func (c *Operator) flowForDeployment(obj interface{}) *v1.ConfigMap {
	key, ok := c.keyFunc(obj)
	if !ok {
		return nil
	}
	// Namespace/Name are one-to-one so the key will find the respective Funktion resource.
	k, exists, err := c.flowInf.GetStore().GetByKey(key)
	if err != nil {
		c.logger.Log("msg", "Flow lookup failed", "err", err)
		return nil
	}
	if !exists {
		return nil
	}
	return k.(*v1.ConfigMap)
}

func (c *Operator) functionForService(obj interface{}) *v1.ConfigMap {
	key, ok := c.keyFunc(obj)
	if !ok {
		return nil
	}
	// Namespace/Name are one-to-one so the key will find the respective Funktion resource.
	k, exists, err := c.functionInf.GetStore().GetByKey(key)
	if err != nil {
		c.logger.Log("msg", "Function lookup failed", "err", err)
		return nil
	}
	if !exists {
		return nil
	}
	return k.(*v1.ConfigMap)
}

func (c *Operator) sync(resourceKey *ResourceKey) error {
	kind := resourceKey.Kind
	key := resourceKey.Key
	c.logger.Log("msg", "reconcile funktion", "key", key, "kind", kind)

	switch kind {
	case FlowKind:
		return c.syncFlow(key)
	case ConnectorKind:
		return nil
	case RuntimeKind:
		return nil
	case FunctionKind:
		return c.syncFunction(key)
	case DeploymentKind:
		return nil
	case ServiceKind:
		return nil
	default:
		c.logger.Log("msg", "Unknown kind funktion", "key", key, "kind", kind)
		return fmt.Errorf("Unknown kind %s for key %s", kind, key)
	}
}

func (c *Operator) syncFlow(key string) error {
	obj, exists, err := c.flowInf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return c.destroyDeployment(key)
	}
	flow := obj.(*v1.ConfigMap)

	connectorName := flow.Labels[ConnectorLabel]
	if len(connectorName) == 0 {
		return fmt.Errorf("Flow %s/%s does not have label %s", flow.Namespace, flow.Name, ConnectorLabel)
	}
	ns := flow.Namespace
	connectorKey := connectorName
	if len(ns) > 0 && !strings.Contains(connectorName, "/") {
		connectorKey = ns + "/" + connectorKey
	}
	obj, exists, err = c.connectorInf.GetIndexer().GetByKey(connectorKey)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Connector %s does not exist for Flow %s/%s current connector keys are %v", connectorKey, flow.Namespace, flow.Name, c.connectorInf.GetIndexer().ListKeys())
	}
	connector := obj.(*v1.ConfigMap)
	if connector == nil {
		return fmt.Errorf("Connector %s does not exist for Flow %s/%s", connectorKey, flow.Namespace, flow.Name)
	}

	deploymentClient := c.kclient.Extensions().Deployments(flow.Namespace)
	obj, exists, err = c.deploymentInf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		d, err := makeFlowDeployment(flow, connector, nil)
		if err != nil {
			return fmt.Errorf("make deployment: %s", err)
		}
		if _, err := deploymentClient.Create(d); err != nil {
			return fmt.Errorf("create deployment: %s", err)
		}
		return nil
	}
	d, err := makeFlowDeployment(flow, connector, obj.(*v1beta1.Deployment))
	if err != nil {
		return fmt.Errorf("update deployment: %s", err)
	}
	if _, err := deploymentClient.Update(d); err != nil {
		return err
	}
	return nil
}

func (c *Operator) destroyDeployment(key string) error {
	obj, exists, err := c.deploymentInf.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	deployment := obj.(*v1beta1.Deployment)

	// Only necessary until GC is properly implemented in both Kubernetes and OpenShift.
	scaleClient := c.kclient.Extensions().Scales(deployment.Namespace)
	if _, err := scaleClient.Update("deployment", &v1beta1.Scale{
		ObjectMeta: v1.ObjectMeta{
			Namespace: deployment.Namespace,
			Name:      deployment.Name,
		},
		Spec: v1beta1.ScaleSpec{
			Replicas: 0,
		},
	}); err != nil {
		return err
	}

	deploymentClient := c.kclient.Extensions().Deployments(deployment.Namespace)
	currentGeneration := deployment.Generation
	if err := wait.PollInfinite(1*time.Second, func() (bool, error) {
		updatedDeployment, err := deploymentClient.Get(deployment.Name)
		if err != nil {
			return false, err
		}
		return updatedDeployment.Status.ObservedGeneration >= currentGeneration &&
			updatedDeployment.Status.Replicas == 0, nil
	}); err != nil {
		return err
	}

	// Let's get ready for proper GC by ensuring orphans are not left behind.
	orphan := false
	return deploymentClient.Delete(deployment.ObjectMeta.Name, &api.DeleteOptions{OrphanDependents: &orphan})
}

func (c *Operator) destroyService(key string) error {
	obj, exists, err := c.serviceInf.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	service := obj.(*v1.Service)

	serviceClient := c.kclient.Services(service.Namespace)
	// Let's get ready for proper GC by ensuring orphans are not left behind.
	orphan := false
	return serviceClient.Delete(service.ObjectMeta.Name, &api.DeleteOptions{OrphanDependents: &orphan})
}

func (c *Operator) syncFunction(key string) error {
	obj, exists, err := c.functionInf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		err = c.destroyDeployment(key)
		if err != nil {
			return err
		}
		return c.destroyService(key)
	}
	function := obj.(*v1.ConfigMap)

	runtimeName := function.Labels[RuntimeLabel]
	if len(runtimeName) == 0 {
		return fmt.Errorf("Function %s/%s does not have label %s", function.Namespace, function.Name, RuntimeLabel)
	}
	ns := function.Namespace
	runtimeKey := runtimeName
	if len(ns) > 0 && !strings.Contains(runtimeName, "/") {
		runtimeKey = ns + "/" + runtimeKey
	}
	obj, exists, err = c.runtimeInf.GetIndexer().GetByKey(runtimeKey)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Runtime %s does not exist for Function %s/%s current connector keys are %v", runtimeKey, function.Namespace, function.Name, c.connectorInf.GetIndexer().ListKeys())
	}
	runtime := obj.(*v1.ConfigMap)
	if runtime == nil {
		return fmt.Errorf("Runtime %s does not exist for Function %s/%s", runtimeKey, function.Namespace, function.Name)
	}

	deploymentClient := c.kclient.Extensions().Deployments(function.Namespace)
	obj, exists, err = c.deploymentInf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	var d2 *v1beta1.Deployment
	if !exists {
		d, err := makeFunctionDeployment(function, runtime, nil)
		if err != nil {
			return fmt.Errorf("make deployment: %s", err)
		}
		if d2, err = deploymentClient.Create(d); err != nil {
			return fmt.Errorf("create deployment: %s", err)
		}
	} else {
		d, err := makeFunctionDeployment(function, runtime, obj.(*v1beta1.Deployment))
		if err != nil {
			return fmt.Errorf("update deployment: %s", err)
		}
		if d2, err = deploymentClient.Update(d); err != nil {
			return err
		}
	}
	serviceClient := c.kclient.Services(function.Namespace)
	obj, exists, err = c.serviceInf.GetIndexer().GetByKey(key)
	if err != nil {
		c.logger.Log("msg", "==== failed to find service", "key", key)
		return err
	}

	if !exists {
		s, err := makeFunctionService(function, runtime, nil, d2)
		if err != nil {
			return fmt.Errorf("make service: %s", err)
		}
		if _, err := serviceClient.Create(s); err != nil {
			return fmt.Errorf("create service: %s", err)
		}
		return nil
	}
	old := obj.(*v1.Service)
	s, err := makeFunctionService(function, runtime, old, d2)
	if err != nil {
		return fmt.Errorf("update service: %s", err)
	}

	// lets copy any missing annotations
	if old.Annotations != nil {
		for k, v := range old.Annotations {
			if len(s.Annotations[k]) == 0 {
				s.Annotations[k] = v
			}
		}
	}
	// lets copy across any missing NodePorts
	s.Spec.Type = old.Spec.Type
	oldPortCount := len(old.Spec.Ports)
	for i, _ := range s.Spec.Ports {
		if i < oldPortCount {
			s.Spec.Ports[i].NodePort = old.Spec.Ports[i].NodePort
		}
	}

	// we must update these fields to be able to update the resource
	s.ResourceVersion = old.ResourceVersion
	s.Spec.ClusterIP = old.Spec.ClusterIP

	if _, err := serviceClient.Update(s); err != nil {
		c.logger.Log("msg", "failed to update service", "name", s.Name, "namespace", function.Namespace)
		return err
	}
	return nil
}
