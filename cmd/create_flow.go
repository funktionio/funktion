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
	"log"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"

	"github.com/funktionio/funktion/pkg/funktion"
	"github.com/funktionio/funktion/pkg/spec"
)

const (
	functionArgPrefix   = "fn:"
	setBodyArgPrefix    = "setBody:"
	setHeadersArgPrefix = "setHeaders:"
)

type createCmdCommon struct {
	kubeclient     *kubernetes.Clientset
	kubeConfigPath string
	namespace      string
	cmd            *cobra.Command
}

type createFlowCmd struct {
	createCmdCommon
	flowName      string
	connectorName string
	args          []string
	trace         bool
	logResult     bool
}

func newCreateFlowCmd() *cobra.Command {
	p := &createFlowCmd{}
	cmd := &cobra.Command{
		Use:   "flow [flags] [endpointUrl] [fn:name] [setBody:content] [setHeaders:foo:bar,xyz:abc]",
		Short: "Creates a new flow which creates an event stream and then invokes a function or HTTP endpoint",
		Long:  `This command will create a new Flow which receives input events and then invokes either a function or HTTP endpoint`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			p.args = args
			err := createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
			if err != nil {
				handleError(err)
				return
			}
			handleError(p.run())
		},
	}
	f := cmd.Flags()
	f.StringVarP(&p.flowName, "name", "n", "", "name of the flow to create")
	f.StringVarP(&p.connectorName, "connector", "c", "", "the Connector name to use. If not specified uses the first URL scheme")
	f.BoolVar(&p.trace, "trace", false, "enable tracing on the flow")
	f.BoolVar(&p.logResult, "log-result", true, "whether to log the result of the subcription to the log of the subcription pod")
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	f.StringVar(&p.namespace, "namespace", "", "the namespace to create the flow inside")
	return cmd
}

func (p *createFlowCmd) run() error {
	var err error
	args := p.args
	if len(args) == 0 {
		return fmt.Errorf("No arguments specified! A flow must have one or more arguments of the form: [endpointUrl] | [function:name] | [setBody:content] | [setHeaders:foo=bar,abc=123]")
	}
	steps, err := parseSteps(args)
	if err != nil {
		return err
	}
	name := p.flowName
	if len(name) == 0 {
		name, err = p.generateName(steps)
		if err != nil {
			return err
		}
	}
	connectorName := p.connectorName
	if len(connectorName) == 0 {
		for _, step := range steps {
			uri := step.URI
			if len(uri) > 0 {
				connectorName, err = urlScheme(uri)
				if err != nil {
					return err
				}
				if len(connectorName) == 0 {
					return fmt.Errorf("No scheme specified for from URI %s", uri)
				}
			}
		}
	}
	funktionConfig := spec.FunkionConfig{
		Flows: []spec.FunktionFlow{
			{
				Name:      "default",
				LogResult: p.logResult,
				Trace:     p.trace,
				Steps:     steps,
			},
		},
	}
	funktionData, err := yaml.Marshal(&funktionConfig)
	if err != nil {
		return fmt.Errorf("Failed to marshal funktion %v due to marshalling error %v", &funktionConfig, err)
	}
	funktionYml := string(funktionData)

	message := stepsText(steps)
	return p.applyFlowWithConnector(name, funktionYml, connectorName, message)
}

func (p *createCmdCommon) applyFlow(fileName, source string) error {
	_, name := filepath.Split(fileName)
	name = convertToSafeResourceName(name[0 : len(name)-len(flowExtension)])
	if len(name) == 0 {
		return fmt.Errorf("Could not generate a name of the flow from file %s", fileName)
	}
	message := fmt.Sprintf("from file %s", fileName)
	// TODO parse from the steps!
	connectorName := "timer"
	return p.applyFlowWithConnector(name, source, connectorName, message)
}

func (p *createCmdCommon) applyFlowWithConnector(name, funktionYml, connectorName, message string) error {
	connector, err := p.checkConnectorExists(connectorName)
	if err != nil {
		return err
	}

	applicationProperties := ""
	if connector.Data != nil {
		applicationProperties = connector.Data[funktion.ApplicationPropertiesProperty]
	}
	if len(applicationProperties) == 0 {
		applicationProperties = "# put your spring boot configuration properties here..."
	}

	labels := map[string]string{
		funktion.KindLabel:      funktion.FlowKind,
		funktion.ConnectorLabel: connectorName,
	}
	data := map[string]string{
		funktion.FunktionYmlProperty:           funktionYml,
		funktion.ApplicationPropertiesProperty: applicationProperties,
	}
	cm := v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: p.namespace,
			Labels:    labels,
		},
		Data: data,
	}
	update := false
	old, err := p.kubeclient.ConfigMaps(p.namespace).Get(name)
	if err == nil {
		update = true
	}

	action := "created"
	if update {
		if old.Data != nil && old.Labels != nil &&
			old.Data[funktion.FunktionYmlProperty] == cm.Data[funktion.FunktionYmlProperty] &&
			old.Data[funktion.ApplicationPropertiesProperty] == cm.Data[funktion.ApplicationPropertiesProperty] &&
			old.Labels[funktion.ConnectorLabel] == cm.Labels[funktion.ConnectorLabel] {
			// source not changed so lets not update!
			return nil
		}
		_, err = p.kubeclient.ConfigMaps(p.namespace).Update(&cm)
		action = "updated"
	} else {
		_, err = p.kubeclient.ConfigMaps(p.namespace).Create(&cm)
	}

	if err == nil {
		log.Println("Flow", name, action, message)
	}
	return err
}

// parseSteps parses a sequence of arguments as either endpoint URLs, function:name,
// setBody:content, setHeaders:foo=bar,abc=def
func parseSteps(args []string) ([]spec.FunktionStep, error) {
	steps := []spec.FunktionStep{}
	for _, arg := range args {
		var step *spec.FunktionStep
		if strings.HasPrefix(arg, functionArgPrefix) {
			name := strings.TrimPrefix(arg, functionArgPrefix)
			if len(name) == 0 {
				return steps, fmt.Errorf("Function name required after %s", functionArgPrefix)
			}
			step = &spec.FunktionStep{
				Kind: spec.FunctionKind,
				Name: name,
			}
		} else if strings.HasPrefix(arg, setBodyArgPrefix) {
			body := strings.TrimPrefix(arg, setBodyArgPrefix)
			step = &spec.FunktionStep{
				Kind: spec.SetBodyKind,
				Body: body,
			}
		} else if strings.HasPrefix(arg, setHeadersArgPrefix) {
			headersText := strings.TrimPrefix(arg, setHeadersArgPrefix)
			if len(headersText) == 0 {
				return steps, fmt.Errorf("Header name and values required after %s", setHeadersArgPrefix)
			}
			headers, err := parseHeaders(headersText)
			if err != nil {
				return steps, err
			}
			step = &spec.FunktionStep{
				Kind:    spec.SetHeadersKind,
				Headers: headers,
			}
		} else {
			step = &spec.FunktionStep{
				Kind: spec.EndpointKind,
				URI:  arg,
			}
		}
		if step != nil {
			steps = append(steps, *step)
		}
	}
	return steps, nil
}

func parseHeaders(text string) (map[string]string, error) {
	m := map[string]string{}
	kvs := strings.Split(text, ",")
	for _, kv := range kvs {
		v := strings.SplitN(kv, ":", 2)
		if len(v) != 2 {
			return m, fmt.Errorf("Missing ':' in header `%s`", kv)
		}
		m[v[0]] = v[1]
	}
	return m, nil
}

func (p *createCmdCommon) checkConnectorExists(name string) (*v1.ConfigMap, error) {
	listOpts, err := funktion.CreateConnectorListOptions()
	if err != nil {
		return nil, err
	}
	cms := p.kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return nil, err
	}
	for _, resource := range resources.Items {
		if resource.Name == name {
			return &resource, nil
		}
	}
	return nil, fmt.Errorf("Connector \"%s\" not found so cannot create this flow", name)
}

func (p *createFlowCmd) generateName(steps []spec.FunktionStep) (string, error) {
	configmaps := p.kubeclient.ConfigMaps(p.namespace)
	cms, err := configmaps.List(api.ListOptions{})
	if err != nil {
		return "", err
	}
	names := make(map[string]bool)
	for _, item := range cms.Items {
		names[item.Name] = true
	}
	prefix := "flow"

	fromUri := ""
	for _, step := range steps {
		fromUri = step.URI
		if len(fromUri) > 0 {
			break
		}
	}

	// lets try generate a flow name from the scheme
	if len(fromUri) > 0 {
		u, err := url.Parse(fromUri)
		if err != nil {
			fmt.Printf("Warning: cannot parse the from URL %s as got %v\n", fromUri, err)
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
	lastCharValid := false
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

// convertToSafeLabelValue converts the given text into a usable kubernetes label value
// removing any dodgy characters
func convertToSafeLabelValue(text string) string {
	var buffer bytes.Buffer
	l := len(text) - 1
	lastCharValid := false
	for i := 0; i <= l; i++ {
		ch := text[i]
		valid := (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if i > 0 && i < l {
			valid = valid || (ch == '-' || ch == '_' || ch == '.')
		}
		if valid {
			buffer.WriteString(string(ch))
			lastCharValid = true
		} else {
			if lastCharValid && i < l {
				buffer.WriteString("-")
			}
			lastCharValid = false
		}
	}
	return buffer.String()
}
