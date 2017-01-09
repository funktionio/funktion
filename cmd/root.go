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
	"fmt"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/tools/clientcmd"
	"github.com/funktionio/funktion/pkg/funktion"
	"github.com/spf13/cobra"
	"k8s.io/client-go/1.5/dynamic"
)

const (
	flowKind = "flow"
	connectorKind = "connector"
	runtimeKind = "runtime"
	functionKind = "function"
)

var RootCmd = &cobra.Command{
	Use:   "funktion",
	Short: "funktion is a Function as a Service (Lambda) style programming model for Kubernetes",
	Long: `Funktion lets you develop complex applications using simple functions and then bind those functions to any event source and run and scale your functions on top of kubernetes.
For more documentation please see: https://funktion.fabric8.io/`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
		}
	},
}

func createKubernetesClient(cmd *cobra.Command, kubeConfigPath string, kubeclientHolder **kubernetes.Clientset, namespace *string) error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if len(kubeConfigPath) > 0 {
		loadingRules.ExplicitPath = kubeConfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	//overrideFlags := clientcmd.RecommendedConfigOverrideFlags("")
	//clientcmd.BindOverrideFlags(overrides, cmd.Flags(), overrideFlags)


	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	cfg, err := kubeConfig.ClientConfig()
	if err != nil {
		fmt.Printf("failed to create Kubernetes client config due to %v\n", err)
		return err
	}
	kubeclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	*kubeclientHolder = kubeclient
	if len(*namespace) == 0 {
		ns, _, err := kubeConfig.Namespace()
		if err != nil {
			return fmt.Errorf("Could not deduce default namespace due to: %v", err)
		}
		*namespace = ns
	}
	return nil
}

func createKubernetesDynamicClient(kubeConfigPath string) (*dynamic.Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if len(kubeConfigPath) > 0 {
		loadingRules.ExplicitPath = kubeConfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	//overrideFlags := clientcmd.RecommendedConfigOverrideFlags("")
	//clientcmd.BindOverrideFlags(overrides, cmd.Flags(), overrideFlags)


	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	cfg, err := kubeConfig.ClientConfig()
	if err != nil {
		fmt.Printf("failed to create Kubernetes client config due to %v\n", err)
		return nil, err
	}
	return dynamic.NewClient(cfg)
}

func handleError(err error) {
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	}
}

func listOptsForKind(kind string) (string, *api.ListOptions, error) {
	switch kind {
	case "flow", "flows":
		listOpts, err := funktion.CreateFlowListOptions()
		return flowKind, listOpts, err
	case "conn", "connector", "connectors":
		listOpts, err := funktion.CreateConnectorListOptions()
		return connectorKind, listOpts, err
	case "r", "runtime", "runtimes":
		listOpts, err := funktion.CreateRuntimeListOptions()
		return runtimeKind, listOpts, err
	case "fn", "function", "functions", "funktion", "funktions":
		listOpts, err := funktion.CreateFunctionListOptions()
		return functionKind, listOpts, err
	default:
		return "", nil, fmt.Errorf("Unknown kind `%s` when known kinds are (`fn`, `flow`, `connector`, `runtime`)", kind)
	}
}


func nameForDeployment(kube *kubernetes.Clientset, namespace string, kind string, name string) (string, error) {
	// TODO we may need to map a function or flow to a different named resource if we have a naming clash
	// so we may need to look at a label or annotation on the function / flow
	return name, nil
}
func nameForService(kube *kubernetes.Clientset, namespace string, kind string, name string) (string, error) {
	// TODO we may need to map a function or flow to a different named resource if we have a naming clash
	// so we may need to look at a label or annotation on the function / flow
	return name, nil
}