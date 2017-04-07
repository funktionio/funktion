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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/funktionio/funktion/pkg/funktion"
	"github.com/funktionio/funktion/pkg/k8sutil"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
)

const (
	chromeDevToolsURLPrefix = "chrome-devtools:"
)

type debugCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	namespace              string
	kind                   string
	name                   string
	localPort              int
	remotePort             int
	supportsChromeDevTools bool
	chromeDevTools         bool
	portText               string

	podAction k8sutil.PodAction
	debugCmd  *exec.Cmd
}

func init() {
	RootCmd.AddCommand(newDebugCmd())
}

func newDebugCmd() *cobra.Command {
	p := &debugCmd{}
	p.podAction = k8sutil.PodAction{
		OnPodChange: p.viewLog,
	}
	cmd := &cobra.Command{
		Use:   "debug KIND NAME [flags]",
		Short: "debugs the given function or flow",
		Long:  `This command will debug the latest container implementing the function or flow`,
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
	f.IntVarP(&p.localPort, "local-port", "l", 0, "The localhost port to use for debugging or the container's debugging port is used")
	f.IntVarP(&p.remotePort, "remote-port", "r", 0, "The remote container port to use for debugging or the container's debugging port is used")
	//f.BoolVarP(&p.chromeDevTools, "chrome", "c", false, "For node based containers open the Chrome DevTools to debug")
	return cmd
}

func (p *debugCmd) run() error {
	name, err := nameForDeployment(p.kubeclient, p.namespace, p.kind, p.name)
	if err != nil {
		return err
	}
	portText, err := p.createPortText(p.kind, p.name)
	if err != nil {
		return err
	}
	p.portText = portText
	kubeclient := p.kubeclient
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

func (p *debugCmd) createPortText(kindText, name string) (string, error) {
	kind, listOpts, err := listOptsForKind(kindText)
	if err != nil {
		return "", err
	}
	cms := p.kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return "", err
	}
	var found *v1.ConfigMap
	for _, resource := range resources.Items {
		if name == resource.Name {
			found = &resource
		}
	}
	if found == nil {
		return "", fmt.Errorf("No %s resource found for name %s", kind, name)
	}

	debugPort := 0
	if kind == functionKind {
		// ensure debug mode is enabled on the function
		if found.Data == nil {
			found.Data = map[string]string{}
		}
		data := found.Data
		debugMode := data[funktion.DebugProperty]
		if strings.ToLower(debugMode) != "true" {
			data[funktion.DebugProperty] = "true"
			_, err = cms.Update(found)
			if err != nil {
				return "", fmt.Errorf("Failed to update Function %s to enable debug mode %v", name, err)
			}
			fmt.Printf("Enabled debug mode for Function %s\n", name)
		}

		// lets use the debug port on the runtime
		runtime := ""
		labels := found.Labels
		if labels != nil {
			runtime = labels[funktion.RuntimeLabel]
		}
		if len(runtime) > 0 {
			return p.createPortText(runtimeKind, runtime)
		}
	} else if kind == runtimeKind || kind == connectorKind {
		data := found.Data
		if data != nil {
			portValue := data[funktion.DebugPortProperty]
			if len(portValue) > 0 {
				debugPort, err = strconv.Atoi(portValue)
				if err != nil {
					return "", fmt.Errorf("Failed to convert debug port `%s` to a number due to %v", portValue, err)
				}
			}
		}
		annotations := found.Annotations
		if kind == runtimeKind && annotations != nil {
			flag := annotations[funktion.ChromeDevToolsAnnotation]
			if flag == "true" {
				p.supportsChromeDevTools = true
			} else if len(flag) == 0 {
				// TODO handle older nodejs runtimes which don't have the annotation
				// remove after next funktion-connectors release!
				if found.Name == "nodejs" {
					p.supportsChromeDevTools = true
				}
			}
		}
	} else if kind == flowKind {
		connector := ""
		data := found.Labels
		if data != nil {
			connector = data[funktion.ConnectorLabel]
		}
		if len(connector) > 0 {
			return p.createPortText(runtimeKind, connector)
		}
	}
	if debugPort == 0 {
		if kind == connectorKind || kind == flowKind {
			// default to java debug port for flows and connectors if none specified
			debugPort = 5005
		}
	}
	if debugPort > 0 {
		if p.localPort == 0 {
			p.localPort = debugPort
		}
		if p.remotePort == 0 {
			p.remotePort = debugPort
		}
	}
	if p.remotePort == 0 {
		return "", fmt.Errorf("No remote debug port could be defaulted. Please specify one via the `-r` flag")
	}
	if p.localPort == 0 {
		p.localPort = p.remotePort
	}
	return fmt.Sprintf("%d:%d", p.localPort, p.remotePort), nil
}

func (p *debugCmd) viewLog(pod *v1.Pod) error {
	if pod != nil {
		binaryFile, err := k8sutil.ResolveKubectlBinary(p.kubeclient)
		if err != nil {
			return err
		}
		name := pod.Name
		if p.debugCmd != nil {
			process := p.debugCmd.Process
			if process != nil {
				process.Kill()
			}
		}
		args := []string{"port-forward", name, p.portText}

		fmt.Printf("\n%s %s\n\n", filepath.Base(binaryFile), strings.Join(args, " "))
		cmd := exec.Command(binaryFile, args...)
		p.debugCmd = cmd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			return err
		}

		if p.supportsChromeDevTools {
			return p.findChromeDevToolsURL(pod, binaryFile)
		}
	}
	return nil
}

func (p *debugCmd) findChromeDevToolsURL(pod *v1.Pod, binaryFile string) error {
	name := pod.Name

	args := []string{"logs", "-f", name}
	cmd := exec.Command(binaryFile, args...)

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout for command %s %s: %v", filepath.Base(binaryFile), strings.Join(args, " "), err)
	}

	scanner := bufio.NewScanner(cmdReader)
	line := 0
	go func() {
		for scanner.Scan() {
			if line++; line > 50 {
				fmt.Printf("No log line found starting with `%s` in the first %d lines. Maybe debug is not really enabled in this pod?\n", chromeDevToolsURLPrefix, line)
				cmdReader.Close()
				killCmd(cmd)
			}
			text := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(text, chromeDevToolsURLPrefix) {
				fmt.Printf("\nTo Debug open: %s\n\n", text)
				cmdReader.Close()
				killCmd(cmd)
				if p.chromeDevTools {
					browser.OpenURL(text)
				}
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		killCmd(cmd)
		return fmt.Errorf("failed to start command %s %s: %v", filepath.Base(binaryFile), strings.Join(args, " "), err)
	}

	err = cmd.Wait()
	if err != nil {
		killCmd(cmd)
		// ignore errors as we get an error if we kill it
	}
	return nil
}

func killCmd(cmd *exec.Cmd) {
	if cmd != nil {
		p := cmd.Process
		if p != nil {
			p.Kill()
		}
	}
}
