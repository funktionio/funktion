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

package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

const (
	exposeURLAnnotation = "fabric8.io/exposeUrl"
)

type locationCom struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	namespace string
	kind      string
	name      string
	open      bool
	retry     bool
}

func init() {
	RootCmd.AddCommand(newLocationCom())
}

func newLocationCom() *cobra.Command {
	p := &locationCom{}
	cmd := &cobra.Command{
		Use:   "url KIND NAME [flags]",
		Short: "views the external URL you can use to access a function or flow",
		Long:  `This command will output the URL you can use to invoke a function of flow`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			if len(args) < 1 {
				handleError(fmt.Errorf("No resource kind argument supplied! Possible values ['fn', 'flow']"))
				return
			}
			if len(args) < 2 {
				handleError(fmt.Errorf("No name specified!"))
				return
			}
			p.kind = args[0]
			p.name = args[1]
			err := createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
			if err != nil {
				handleError(err)
				return
			}
			handleError(p.run())
		},
	}
	f := cmd.Flags()
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	f.StringVarP(&p.namespace, "namespace", "n", "", "the namespace to query")
	f.StringVarP(&p.name, "name", "v", "latest", "the version of the connectors to install")
	f.BoolVarP(&p.open, "open", "o", false, "Open the URL in a browser")
	f.BoolVarP(&p.retry, "retry", "r", true, "Whether to keep retrying if the endpoint is not yet available")
	return cmd
}

func (p *locationCom) run() error {
	name, err := nameForService(p.kubeclient, p.namespace, p.kind, p.name)
	if err != nil {
		return err
	}
	return p.openService(name)
}

func (p *locationCom) openService(serviceName string) error {
	c := p.kubeclient
	ns := p.namespace
	if p.retry {
		if err := RetryAfter(40, func() error {
			return CheckService(c, ns, serviceName)
		}, 10*time.Second); err != nil {
			fmt.Errorf("Could not find finalized endpoint being pointed to by %s: %v", serviceName, err)
			os.Exit(1)
		}
	}
	svcs, err := c.Services(ns).List(api.ListOptions{})
	if err != nil {
		return fmt.Errorf("No services found %v\n", err)
	}
	for _, service := range svcs.Items {
		if serviceName == service.Name {
			url := service.ObjectMeta.Annotations[exposeURLAnnotation]
			if p.open {
				fmt.Printf("\nOpening URL %s\n", url)
				browser.OpenURL(url)
			} else {
				fmt.Printf("%s\n", url)
			}
			return nil
		}
	}
	return fmt.Errorf("No service %s in namespace %s\n", serviceName, ns)
}

// CheckService waits for the specified service to be ready by returning an error until the service is up
// The check is done by polling the endpoint associated with the service and when the endpoint exists, returning no error->service-online
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go#L89
func CheckService(c *kubernetes.Clientset, ns string, serviceName string) error {
	svc, err := c.Services(ns).Get(serviceName)
	if err != nil {
		return err
	}
	url := svc.ObjectMeta.Annotations[exposeURLAnnotation]
	if url == "" {
		fmt.Print(".")
		return errors.New("")
	}
	endpoints := c.Endpoints(ns)
	if endpoints == nil {
		fmt.Errorf("No endpoints found in namespace %s\n", ns)
	}
	endpoint, err := endpoints.Get(serviceName)
	if err != nil {
		fmt.Errorf("No endpoints found for service %s\n", serviceName)
		return err
	}
	return CheckEndpointReady(endpoint)
}

//CheckEndpointReady checks that the kubernetes endpoint is ready
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go#L101
func CheckEndpointReady(endpoint *v1.Endpoints) error {
	if len(endpoint.Subsets) == 0 {
		fmt.Fprintf(os.Stderr, ".")
		return fmt.Errorf("Endpoint for service is not ready yet\n")
	}
	for _, subset := range endpoint.Subsets {
		if len(subset.NotReadyAddresses) != 0 {
			fmt.Fprintf(os.Stderr, "Waiting, endpoint for service is not ready yet...\n")
			return fmt.Errorf("Endpoint for service is not ready yet\n")
		}
	}
	return nil
}

func Retry(attempts int, callback func() error) (err error) {
	return RetryAfter(attempts, callback, 0)
}

func RetryAfter(attempts int, callback func() error, d time.Duration) (err error) {
	m := MultiError{}
	for i := 0; i < attempts; i++ {
		err = callback()
		if err == nil {
			return nil
		}
		m.Collect(err)
		time.Sleep(d)
	}
	return m.ToError()
}

type MultiError struct {
	Errors []error
}

func (m *MultiError) Collect(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

func (m MultiError) ToError() error {
	if len(m.Errors) == 0 {
		return nil
	}

	errStrings := []string{}
	for _, err := range m.Errors {
		errStrings = append(errStrings, err.Error())
	}
	return fmt.Errorf(strings.Join(errStrings, "\n"))
}
