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
	"strconv"
	"net/url"
	"strings"
	"bytes"
)

type subscribeCmd struct {
	subscriptionName string
	connectorName    string
	fromUrl          string
	toUrl            string
	namespace        string
	kubeConfigPath   string
	cmd              *cobra.Command
	kubeclient       *kubernetes.Clientset
}

func init() {
	RootCmd.AddCommand(newSubscribeCmd())
}

func newSubscribeCmd() *cobra.Command {
	p := &subscribeCmd{
	}
	cmd := &cobra.Command{
		Use:   "subscribe",
		Short: "Subscribes to the given event and then invokes a function or HTTP endpoint",
		Long:  `This command will create a new Subscription which receives input events and then invokes either a function or HTTP endpoint`,
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
	f.StringVarP(&p.subscriptionName, "name", "n", "", "name of the subscription to create")
	f.StringVarP(&p.fromUrl, "from", "f", "", "the URL to consume from")
	f.StringVarP(&p.toUrl, "to", "t", "", "the URL to invoke")
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	return cmd
}

func (p *subscribeCmd) run() error {
	var err error
	name := p.subscriptionName
	if len(name) == 0 {
		name, err = p.generateName()
		if err != nil {
			return err
		}
	}
	fmt.Printf("Generating subscription %s from %s to %s\n", name, p.fromUrl, p.toUrl)
	return nil
}

func (p *subscribeCmd) generateName() (string, error) {
	configmaps := p.kubeclient.ConfigMaps(p.namespace)
	cms, err := configmaps.List(api.ListOptions{})
	if err != nil {
		return "", err
	}
	names := make(map[string]bool)
	for _, item := range cms.Items {
		names[item.Name] = true
	}
	prefix := "subscription"

	// lets try generate a subscription name from the scheme
	if len(p.fromUrl) > 0 {
		u, err := url.Parse(p.fromUrl)
		if err != nil {
			fmt.Printf("Warning: cannot parse the from URL %s as got %v\n", p.fromUrl, err)
		} else {
			path := strings.Trim(u.Host, "/")
			prefix = u.Scheme
			if len(p.connectorName) == 0 {
				p.connectorName = u.Scheme
			}
			if len(path) > 0 {
				prefix = prefix + "-" + path
			}
			prefix = convertToSafeResourceName(prefix)
		}
	}
	counter := 1
	for {
		name := prefix + strconv.Itoa(counter)
		if !names[name] {
			return name, nil
		}
		counter++
	}

}

// convertToSafeResourceName converts the given text into a usable kubernetes name
// converting to lower case and removing any dodgy characters
func convertToSafeResourceName(text string) string {
	var buffer bytes.Buffer
	lower := strings.ToLower(text)
	lastCharValid := false;
	for i := 0; i < len(lower); i++ {
		ch := lower[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			buffer.WriteString(string(ch))
			lastCharValid = true
		} else {
			if lastCharValid {
				buffer.WriteString("-")
			}
			lastCharValid = false
		}
	}
	return buffer.String()
}