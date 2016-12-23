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
	"io/ioutil"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/fabric8io/funktion-operator/pkg/funktion"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

type createFunctionCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	namespace      string
	name           string
	runtime        string
	source           string
	file           string

	configMaps    map[string]*v1.ConfigMap
}

func init() {
	RootCmd.AddCommand(newCreateCmd())
}

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create KIND [NAME] [flags]",
		Short: "creates a new resources",
		Long:  `This command will create a new resource`,
	}

	subscribeCmd := newSubscribeCmd()
	subscribeCmd.Use = "subscription"
	subscribeCmd.Short = "Creates a Subscription flow"

	cmd.AddCommand(subscribeCmd)
	cmd.AddCommand(newCreateFunctionCmd())
	return cmd
}

func newCreateFunctionCmd() *cobra.Command {
	p := &createFunctionCmd{
	}
	cmd := &cobra.Command{
		Use:   "function [flags]",
		Short: "creates a new function resource",
		Long:  `This command will create a new function resource`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			err := createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
			if err != nil {
				handleError(err)
				return
			}
			handleError(p.run())
		},
	}
	f := cmd.Flags()
	f.StringVarP(&p.name, "name", "n", "", "the name of the function to create")
	f.StringVarP(&p.source, "source", "s", "", "the source code of the function to create")
	f.StringVarP(&p.file, "file", "f", "", "the file name that contains the source code for the function to create")
	f.StringVarP(&p.runtime, "runtime", "r", "nodejs", "the runtime to use. e.g. 'nodejs'")
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	f.StringVar(&p.namespace, "namespace", "", "the namespace to query")
	return cmd
}

func (p *createFunctionCmd) run() error {
	listOpts, err := funktion.CreateFunctionListOptions()
	if err != nil {
		return err
	}
	kubeclient := p.kubeclient
	cms := kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return err
	}
	p.configMaps = map[string]*v1.ConfigMap{}
	for _, resource := range resources.Items {
		p.configMaps[resource.Name] = &resource
	}

	name := p.name
	if len(name) == 0 {
		name, err = p.generateName()
		if err != nil {
			return err
		}
	}
        update := p.configMaps[name] != nil
        cm, err := p.createFunction(name)
	if err != nil {
		return err
	}
	message := "created"
	if update {
		_, err = cms.Update(cm);
		message = "updated"
	} else {
		_, err = cms.Create(cm);
	}
	if err == nil {
		fmt.Printf("Function %s %s\n", name, message)
	}
	return err
}

func (p *createFunctionCmd) generateName() (string, error) {
	prefix := "function"
	counter := 1
	for {
		name := prefix + strconv.Itoa(counter)
		if p.configMaps[name] == nil {
			return name, nil
		}
		counter++
	}
}

func (p *createFunctionCmd) createFunction(name string) (*v1.ConfigMap, error) {
	runtime := p.runtime
	if len(runtime) == 0 {
		return nil, fmt.Errorf("No runtime supplied! Please pass `-n nodejs` or some other valid runtime")
	}
	err := p.checkRuntimeExists(runtime)
	if err != nil {
		return nil, err
	}
	source := p.source
	if len(source) == 0 {
		file := p.file
		if len(file) == 0 {
			return nil, fmt.Errorf("No function source code or file name supplied! You must specify either -s or -f flags")
		}
		source, err = loadFileSource(file)
	}
	cm := &v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				funktion.KindLabel: funktion.FunctionKind,
				funktion.RuntimeLabel: runtime,
			},
		},
		Data: map[string]string{
			funktion.SourceProperty: source,
		},
	}
	return cm, nil
}

func loadFileSource(fileName string) (string, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	source := string(data)
	if len(source) == 0 {
		return "", fmt.Errorf("The file %s is empty!", fileName)
	}
	return source, nil
}

func (p *createFunctionCmd) checkRuntimeExists(name string) error {
	listOpts, err := funktion.CreateRuntimeListOptions()
	if err != nil {
		return err
	}
	cms, err := p.kubeclient.ConfigMaps(p.namespace).List(*listOpts)
	if err != nil {
		return err
	}
	for _, resource := range cms.Items {
		if resource.Name == name {
			return nil
		}
	}
	return fmt.Errorf("No runtime exists called `%s`", name)

}
