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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/funktionio/funktion/pkg/k8sutil"
	"github.com/spf13/cobra"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
)

type logCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	namespace string
	kind      string
	name      string
	follow    bool

	podAction k8sutil.PodAction
	logCmd    *exec.Cmd
}

func init() {
	RootCmd.AddCommand(newLogCmd())
}

func newLogCmd() *cobra.Command {
	p := &logCmd{}
	p.podAction = k8sutil.PodAction{
		OnPodChange: p.viewLog,
	}
	cmd := &cobra.Command{
		Use:   "logs KIND NAME [flags]",
		Short: "tails the log of the given function or flow",
		Long:  `This command will tail the log of the latest container implementing the function or flow`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			if len(args) < 1 {
				handleError(fmt.Errorf("No resource kind argument supplied! Possible values ['fn', 'flow']"))
				return
			}
			p.kind = args[0]
			kind, _, err := listOptsForKind(p.kind)
			if err != nil {
				handleError(err)
				return
			}
			if len(args) < 2 {
				handleError(fmt.Errorf("No %s name specified!", kind))
				return
			}
			p.name = args[1]
			err = createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
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
	f.BoolVarP(&p.follow, "follow", "f", true, "Whether or not to follow the log")
	return cmd
}

func (p *logCmd) run() error {
	kubeclient := p.kubeclient
	name, err := nameForDeployment(p.kubeclient, p.namespace, p.kind, p.name)
	if err != nil {
		return err
	}
	ds, err := kubeclient.Deployments(p.namespace).List(api.ListOptions{})
	if err != nil {
		return err
	}
	var deployment *v1beta1.Deployment
	for _, item := range ds.Items {
		if item.Name == name {
			deployment = &item
			break
		}
	}
	if deployment == nil {
		return fmt.Errorf("No Deployment found called `%s`", name)
	}
	selector := deployment.Spec.Selector
	if selector == nil {
		return fmt.Errorf("Deployment `%s` does not have a selector!", name)
	}
	if selector.MatchLabels == nil {
		return fmt.Errorf("Deployment `%s` selector does not have a matchLabels!", name)
	}
	listOpts, err := k8sutil.V1BetaSelectorToListOptions(selector)
	if err != nil {
		return err
	}
	p.podAction.WatchPods(p.kubeclient, p.namespace, listOpts)
	return p.podAction.WatchLoop()
}

func (p *logCmd) viewLog(pod *v1.Pod) error {
	if pod != nil {
		binaryFile, err := k8sutil.ResolveKubectlBinary(p.kubeclient)
		if err != nil {
			return err
		}
		name := pod.Name
		if p.logCmd != nil {
			process := p.logCmd.Process
			if process != nil {
				process.Kill()
			}
		}
		args := []string{"logs"}
		if p.follow {
			args = append(args, "-f")
		}
		args = append(args, name)

		fmt.Printf("\n%s %s\n\n", filepath.Base(binaryFile), strings.Join(args, " "))
		cmd := exec.Command(binaryFile, args...)
		p.logCmd = cmd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			return err
		}
	}
	return nil
}
