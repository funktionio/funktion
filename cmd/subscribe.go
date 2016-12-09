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
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ghodss/yaml"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"

	"github.com/fabric8io/funktion-operator/pkg/funktion"
	"github.com/fabric8io/funktion-operator/pkg/spec"
)

type subscribeCmd struct {
	subscriptionName string
	connectorName    string
	fromUrl          string
	toUrl            string
	trace            bool
	logResult        bool
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
	f.BoolVar(&p.trace, "trace", false, "enable tracing on the subscription")
	f.BoolVar(&p.logResult, "log-result", true, "whether to log the result of the subcription to the log of the subcription pod")
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	return cmd
}

func (p *subscribeCmd) run() error {
	var err error
	update := false
	name := p.subscriptionName
	if len(name) == 0 {
		name, err = p.generateName()
		if err != nil {
			return err
		}
	} else {
		_, err = p.kubeclient.ConfigMaps(p.namespace).Get(name)
		if err == nil {
			update = true
		}
	}
	fromUrl := p.fromUrl
	toUrl := p.toUrl
	connectorName, err := urlScheme(fromUrl)
	if err != nil {
		return err
	}
	if len(connectorName) == 0 {
		return fmt.Errorf("No scheme specified for from URL %s", fromUrl)
	}
	err = p.checkConnectorExists(connectorName)
	if err != nil {
		return err
	}

	funktionConfig := spec.FunkionConfig{
		Rules: []spec.FunktionRule{
			spec.FunktionRule{
				Name: "default",
				Trigger: fromUrl,
				LogResult: p.logResult,
				Trace: p.trace,
				Actions: []spec.FunktionAction{
					spec.FunktionAction{
						Kind: spec.EndpointKind,
						URL: toUrl,
					},
				},
			},
		},
	}
	funktionData, err := yaml.Marshal(&funktionConfig)
	if err != nil {
		return fmt.Errorf("Failed to marshal funktion %v due to marshalling error %v", &funktionConfig, err)
	}
	funktionYml := string(funktionData)
	applicationProperties := "# put your spring boot configuration properties here..."

	labels := map[string]string{
		funktion.KindLabel: funktion.SubscriptionKind,
		funktion.ConnectorLabel: connectorName,
	}
	data := map[string]string{
		funktion.FunktionYmlProperty: funktionYml,
		funktion.ApplicationPropertiesProperty: applicationProperties,

	}
	cm := v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
			Namespace: p.namespace,
			Labels: labels,
		},
		Data: data,
	}
	if update {
		_, err = p.kubeclient.ConfigMaps(p.namespace).Update(&cm)
	} else {
		_, err = p.kubeclient.ConfigMaps(p.namespace).Create(&cm)
	}
	if err == nil {
		fmt.Printf("Created subscription %s from %s => %s\n", name, fromUrl, toUrl)
	}
	return err
}

func (p *subscribeCmd) checkConnectorExists(name string) error {
	listOpts, err := funktion.CreateConnectorListOptions()
	if err != nil {
		return err
	}
	cms := p.kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return err
	}
	for _, resource := range resources.Items {
		if resource.Name == name {
			return nil
		}
	}
	return fmt.Errorf("Connector \"%s\" not found so cannot create this subscription", name)
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

func urlScheme(text string) (string, error) {
	u, err := url.Parse(text)
	if err != nil {
		return "", fmt.Errorf("Warning: cannot parse the from URL %s as got %v\n", text, err)
	} else {
		return u.Scheme, nil
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