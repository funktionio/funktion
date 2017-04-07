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
	"bytes"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"

	"github.com/funktionio/funktion/pkg/funktion"
	"github.com/funktionio/funktion/pkg/spec"
)

type getCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	kind      string
	namespace string
	name      string

	deployments map[string]*v1beta1.Deployment
	services    map[string]*v1.Service
}

func init() {
	RootCmd.AddCommand(newGetCmd())
}

func newGetCmd() *cobra.Command {
	p := &getCmd{}
	cmd := &cobra.Command{
		Use:   "get KIND [NAME] [flags]",
		Short: "gets a list of the resources",
		Long:  `This command will list all of the resources of a given kind`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			if len(args) == 0 {
				handleError(fmt.Errorf("No resource kind argument supplied! Possible values ['connector', 'flow', 'function', 'runtime']"))
				return
			}
			p.kind = args[0]
			if len(args) > 1 {
				p.name = args[1]
			}
			err := createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
			if err != nil {
				handleError(err)
				return
			}
			handleError(p.run())
		},
	}
	f := cmd.Flags()
	//f.StringVarP(&p.format, "output", "o", "", "The format of the output")
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	f.StringVarP(&p.namespace, "namespace", "n", "", "the namespace to query")
	return cmd
}

func (p *getCmd) run() error {
	kind, listOpts, err := listOptsForKind(p.kind)
	if err != nil {
		return err
	}
	kubeclient := p.kubeclient
	cms := kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return err
	}
	p.deployments = map[string]*v1beta1.Deployment{}
	p.services = map[string]*v1.Service{}
	ds, err := kubeclient.Deployments(p.namespace).List(api.ListOptions{})
	if err != nil {
		return err
	}
	p.deployments = map[string]*v1beta1.Deployment{}
	for _, item := range ds.Items {
		// TODO lets assume the name of the Deployment is the name of the Flow
		// but we may want to use a label instead to link them?
		name := item.Name
		p.deployments[name] = &item
	}
	if kind == functionKind {
		ss, err := kubeclient.Services(p.namespace).List(api.ListOptions{})
		if err != nil {
			return err
		}
		for _, item := range ss.Items {
			// TODO lets assume the name of the Service is the name of the Function
			// but we may want to use a label instead to link them?
			name := item.Name
			p.services[name] = &item
		}
	}
	name := p.name
	if len(name) == 0 {
		p.printHeader(kind)
		for _, resource := range resources.Items {
			p.printResource(&resource, kind)
		}

	} else {
		found := false
		for _, resource := range resources.Items {
			if resource.Name == name {
				p.printHeader(kind)
				p.printResource(&resource, kind)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s \"%s\" not found", kind, name)
		}
	}
	return nil
}

func (p *getCmd) printHeader(kind string) {
	switch kind {
	case flowKind:
		printFlowRow("NAME", "PODS", "STEPS")
	case functionKind:
		printFunctionRow("NAME", "PODS", "URL")
	default:
		printRuntimeRow("NAME", "VERSION")
	}
}

func (p *getCmd) printResource(cm *v1.ConfigMap, kind string) {
	switch kind {
	case functionKind:
		printFunctionRow(cm.Name, p.podText(cm), p.functionURLText(cm))
	case flowKind:
		printFlowRow(cm.Name, p.podText(cm), p.flowStepsText(cm))
	default:
		printRuntimeRow(cm.Name, p.runtimeVersion(cm))
	}
}

func printFunctionRow(name string, pod string, flow string) {
	fmt.Printf("%-32s %-9s %s\n", name, pod, flow)
}

func printFlowRow(name string, pod string, flow string) {
	fmt.Printf("%-32s %-9s %s\n", name, pod, flow)
}

func printRuntimeRow(name string, version string) {
	fmt.Printf("%-32s %s\n", name, version)
}

func (p *getCmd) podText(cm *v1.ConfigMap) string {
	name := cm.Name
	deployment := p.deployments[name]
	if deployment == nil {
		return ""
	}
	var status = deployment.Status
	return fmt.Sprintf("%d/%d", status.AvailableReplicas, status.Replicas)
}

func (p *getCmd) functionURLText(cm *v1.ConfigMap) string {
	name := cm.Name
	service := p.services[name]
	if service == nil || service.Annotations == nil {
		return ""
	}
	return service.Annotations[exposeURLAnnotation]
}

func (p *getCmd) runtimeVersion(cm *v1.ConfigMap) string {
	labels := cm.Labels
	if labels != nil {
		return labels[funktion.VersionLabel]
	}
	return ""
}

func (p *getCmd) flowStepsText(cm *v1.ConfigMap) string {
	yamlText := cm.Data[funktion.FunktionYmlProperty]
	if len(yamlText) == 0 {
		return fmt.Sprintf("No `%s` property specified", funktion.FunktionYmlProperty)
	}
	fc := spec.FunkionConfig{}
	err := yaml.Unmarshal([]byte(yamlText), &fc)
	if err != nil {
		return fmt.Sprintf("Failed to parse `%s` YAML: %v", funktion.FunktionYmlProperty, err)
	}
	if len(fc.Flows) == 0 {
		return "No funktion flows"
	}
	rule := fc.Flows[0]
	return stepsText(rule.Steps)
}

func stepsText(steps []spec.FunktionStep) string {
	actionMessage := "No steps!"
	if len(steps) > 0 {
		var buffer bytes.Buffer
		for i, step := range steps {
			if i > 0 {
				buffer.WriteString(" => ")
			}
			kind := step.Kind
			text := kind
			switch kind {
			case spec.EndpointKind:
				text = fmt.Sprintf("%s", step.URI)
			case spec.FunctionKind:
				text = fmt.Sprintf("function %s", step.Name)
			}
			buffer.WriteString(text)
		}
		actionMessage = buffer.String()
	}
	return actionMessage
}
