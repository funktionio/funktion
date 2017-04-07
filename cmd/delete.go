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
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"strings"
)

type deleteCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	kind      string
	namespace string
	name      string
	all       bool
}

func init() {
	RootCmd.AddCommand(newDeleteCmd())
}

func newDeleteCmd() *cobra.Command {
	p := &deleteCmd{}
	cmd := &cobra.Command{
		Use:   "delete KIND ([NAME] | --all) [flags]",
		Short: "delete resources",
		Long:  `This command will delete one more resources`,
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
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	f.StringVarP(&p.namespace, "namespace", "n", "", "the namespace to query")
	f.BoolVar(&p.all, "all", false, "whether to delete all resources")
	return cmd
}

func (p *deleteCmd) run() error {
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
		if !p.all {
			return fmt.Errorf("No `name` specified or the `--all` flag specified so cannot delete a %s", kind)
		}
		count := 0
		for _, resource := range resources.Items {
			err = p.deleteResource(&resource)
			if err != nil {
				return err
			}
			count++
		}
		fmt.Printf("Deleted %d %s resource(s)\n", count, kind)
	} else {
		for _, resource := range resources.Items {
			if resource.Name == name {
				err = p.deleteResource(&resource)
				if err == nil {
					fmt.Printf("Deleted %s \"%s\" resource\n", kind, name)
				}
				return err
			}
		}
		return fmt.Errorf("%s \"%s\" not found", kind, name)
	}
	return nil
}

func (p *deleteCmd) deleteResource(cm *v1.ConfigMap) error {
	kind := strings.ToLower(cm.Kind)
	name := cm.Name
	ns := cm.Namespace
	if len(ns) == 0 {
		ns = p.namespace
	}
	err := p.kubeclient.ConfigMaps(ns).Delete(name, &api.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Failed to delete %s \"%s\" due to: %v", kind, name, err)
	}
	return nil
}
