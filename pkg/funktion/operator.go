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

	"github.com/fabric8io/funktion-operator/pkg/analytics"
	"github.com/fabric8io/funktion-operator/pkg/queue"

	"github.com/go-kit/kit/log"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	extensionsobj "k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	utilruntime "k8s.io/client-go/1.5/pkg/util/runtime"
	"k8s.io/client-go/1.5/pkg/util/wait"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
	"strings"
)

const (
	ConnectorLabel = "connector"
)

// Operator manages Funktion Deployments
type Operator struct {
	kclient         *kubernetes.Clientset
	//funktionClient *rest.RESTClient
	logger          log.Logger

	connectorInf cache.SharedIndexInformer
	subscriptionInf cache.SharedIndexInformer
	deploymentInf   cache.SharedIndexInformer

	queue           *queue.Queue
}

// New creates a new controller.
func New(cfg *rest.Config, logger log.Logger) (*Operator, error) {
	logger.Log("msg", "starting up!")
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := &Operator{
		kclient:        client,
		logger:         logger,
		queue:          queue.New(),
	}

	logger.Log("msg", "creating ListOptions")
	subscriptionListOpts, err := CreateSubscriptionListOptions()
	if err != nil {
		return nil, err
	}
	connectorListOpts, err := CreateConnectorListOptions()
	if err != nil {
		return nil, err
	}

	c.connectorInf = cache.NewSharedIndexInformer(
		NewConfigMapListWatch(c.kclient, *connectorListOpts),
		&v1.ConfigMap{},
		resyncPeriod,
		cache.Indexers{},
	)
	c.subscriptionInf = cache.NewSharedIndexInformer(
		NewConfigMapListWatch(c.kclient, *subscriptionListOpts),
		&v1.ConfigMap{},
		resyncPeriod,
		cache.Indexers{},
	)
	c.deploymentInf = cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(c.kclient.Extensions().GetRESTClient(), "deployments", api.NamespaceAll, nil),
		&extensionsobj.Deployment{},
		resyncPeriod,
		cache.Indexers{},
	)

	c.connectorInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddConnector,
		DeleteFunc: c.handleDeleteConnector,
		UpdateFunc: c.handleUpdateConnector,
	})
	c.subscriptionInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddSubscription,
		DeleteFunc: c.handleDeleteSubscription,
		UpdateFunc: c.handleUpdateSubscription,
	})
	c.deploymentInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// TODO only look at Funktion related Deployments?
		AddFunc: func(d interface{}) {
			c.logger.Log("msg", "addDeployment", "trigger", "depl add")
			c.handleAddDeployment(d)
		},
		DeleteFunc: func(d interface{}) {
			c.logger.Log("msg", "deleteDeployment", "trigger", "depl delete")
			c.handleDeleteDeployment(d)
		},
		UpdateFunc: func(old, cur interface{}) {
			c.logger.Log("msg", "updateDeployment", "trigger", "depl update")
			c.handleUpdateDeployment(old, cur)
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
	go c.subscriptionInf.Run(stopc)
	go c.deploymentInf.Run(stopc)

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

func (c *Operator) handleAddConnector(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.ConnectorCreated()
	c.logger.Log("msg", "Connector added", "key", key)
	//c.enqueue(key)
}

func (c *Operator) handleDeleteConnector(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.ConnectorDeleted()
	c.logger.Log("msg", "Connector deleted", "key", key)
	//c.enqueue(key)
}

func (c *Operator) handleUpdateConnector(old, cur interface{}) {
	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	c.logger.Log("msg", "Connector updated", "key", key)
	//c.enqueue(key)
}

func (c *Operator) handleAddSubscription(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.SubscriptionCreated()
	c.logger.Log("msg", "Subscription added", "key", key)
	c.enqueue(key)
}

func (c *Operator) handleDeleteSubscription(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.SubscriptionDeleted()
	c.logger.Log("msg", "Subscription deleted", "key", key)
	c.enqueue(key)
}

func (c *Operator) handleUpdateSubscription(old, cur interface{}) {
	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	c.logger.Log("msg", "Subscription updated", "key", key)
	c.enqueue(key)
}

// enqueue adds a key to the queue. If obj is a key already it gets added directly.
// Otherwise, the key is extracted via keyFunc.
func (c *Operator) enqueue(obj interface{}) {
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

	c.queue.Add(key)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (c *Operator) worker() {
	for {
		key, quit := c.queue.Get()
		if quit {
			return
		}
		if err := c.sync(key.(string)); err != nil {
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

func (c *Operator) subscriptionForDeployment(obj interface{}) *v1.ConfigMap {
	key, ok := c.keyFunc(obj)
	if !ok {
		return nil
	}
	// Namespace/Name are one-to-one so the key will find the respective Funktion resource.
	k, exists, err := c.subscriptionInf.GetStore().GetByKey(key)
	if err != nil {
		c.logger.Log("msg", "Funktion lookup failed", "err", err)
		return nil
	}
	if !exists {
		return nil
	}
	return k.(*v1.ConfigMap)
}

func (c *Operator) handleDeleteDeployment(obj interface{}) {
	if d := c.subscriptionForDeployment(obj); d != nil {
		c.enqueue(d)
	}
}

func (c *Operator) handleAddDeployment(obj interface{}) {
	if d := c.subscriptionForDeployment(obj); d != nil {
		c.enqueue(d)
	}
}

func (c *Operator) handleUpdateDeployment(oldo, curo interface{}) {
	old := oldo.(*extensionsobj.Deployment)
	cur := curo.(*extensionsobj.Deployment)

	c.logger.Log("msg", "update handler", "old", old.ResourceVersion, "cur", cur.ResourceVersion)

	// Periodic resync may resend the deployment without changes in-between.
	// Also breaks loops created by updating the resource ourselves.
	if old.ResourceVersion == cur.ResourceVersion {
		return
	}

	// Wake up Funktion resource the deployment belongs to.
	if k := c.subscriptionForDeployment(cur); k != nil {
		c.enqueue(k)
	}
}

func (c *Operator) sync(key string) error {
	c.logger.Log("msg", "reconcile funktion", "key", key)

	obj, exists, err := c.subscriptionInf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return c.destroySubscription(key)
	}
	subscription := obj.(*v1.ConfigMap)
	c.logger.Log("msg", fmt.Sprintf("ConfigMap has name %s labels %v and annotations %v", subscription.Name, subscription.Labels, subscription.Annotations), "key", key)

	connectorName := subscription.Labels[ConnectorLabel]
	if len(connectorName) == 0 {
		return fmt.Errorf("Subscription %s/%s does not have label %s", subscription.Namespace, subscription.Name, ConnectorLabel)
	}
	ns := subscription.Namespace
	connectorKey := connectorName
	if len(ns) > 0 && !strings.Contains(connectorName, "/") {
		connectorKey = ns + "/" + connectorKey
	}
	obj, exists, err = c.connectorInf.GetIndexer().GetByKey(connectorKey)
	if err != nil {
		return err
	}
	if (!exists) {
		return fmt.Errorf("Connector %s does not exist for Subscription %s/%s current connector keys are %v", connectorKey, subscription.Namespace, subscription.Name, c.connectorInf.GetIndexer().ListKeys())
	}
	connector := obj.(*v1.ConfigMap)
	if connector == nil {
		return fmt.Errorf("Connector %s does not exist for Subscription %s/%s", connectorKey, subscription.Namespace, subscription.Name)
	}

	deploymentClient := c.kclient.Extensions().Deployments(subscription.Namespace)
	obj, exists, err = c.deploymentInf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		d, err := makeDeployment(subscription, connector, nil)
		if err != nil {
			return fmt.Errorf("make deployment: %s", err)
		}
		if _, err := deploymentClient.Create(d); err != nil {
			return fmt.Errorf("create deployment: %s", err)
		}
		return nil
	}
	d, err := makeDeployment(subscription, connector, obj.(*extensionsobj.Deployment))
	if err != nil {
		return fmt.Errorf("update deployment: %s", err)
	}
	if _, err := deploymentClient.Update(d); err != nil {
		return err
	}
	return nil
}

func (c *Operator) destroySubscription(key string) error {
	obj, exists, err := c.deploymentInf.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	deployment := obj.(*extensionsobj.Deployment)

	// Only necessary until GC is properly implemented in both Kubernetes and OpenShift.
	scaleClient := c.kclient.Extensions().Scales(deployment.Namespace)
	if _, err := scaleClient.Update("deployment", &extensionsobj.Scale{
		ObjectMeta: v1.ObjectMeta{
			Namespace: deployment.Namespace,
			Name:      deployment.Name,
		},
		Spec: extensionsobj.ScaleSpec{
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