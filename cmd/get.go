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

	"github.com/spf13/cobra"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

type getCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	kind           string
	namespace      string
	name           string
}

func init() {
	RootCmd.AddCommand(newGetCmd())
}

func newGetCmd() *cobra.Command {
	p := &getCmd{
	}
	cmd := &cobra.Command{
		Use:   "get KIND [NAME] [flags]",
		Short: "gets a list of the resources",
		Long:  `This command will list all of the resources of a given kind`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			if len(args) == 0 {
				handleError(fmt.Errorf("No resource kind argument supplied!"))
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
	cms := p.kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return err
	}
	name := p.name
	if len(name) == 0 {
		p.printHeader()
		for _, resource := range resources.Items {
			p.printResource(&resource)
		}

	} else {
		found := false
		for _, resource := range resources.Items {
			if resource.Name == name {
				p.printHeader()
				p.printResource(&resource)
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


func (p *getCmd) printHeader() {
	fmt.Printf("NAME\n")
}

func (p *getCmd) printResource(cm *v1.ConfigMap) {
	fmt.Printf("%s\n", cm.Name)
}
