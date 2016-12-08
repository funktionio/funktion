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
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/fabric8io/funktion-operator/pkg/funktion"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/runtime"
)

const (
	connectorMetadataUrl = "io/fabric8/funktion/connector-package/maven-metadata.xml"
	connectorPackageUrlPrefix = "io/fabric8/funktion/connector-package/%[1]s/connector-package-%[1]s-"
)

type installCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	namespace      string
	names          []string
	version        string
	mavenRepo      string
	replace        bool
	listConnectors bool
}

func init() {
	RootCmd.AddCommand(newInstallCmd())
}

func newInstallCmd() *cobra.Command {
	p := &installCmd{
	}
	cmd := &cobra.Command{
		Use:   "install [NAMES] [flags]",
		Short: "installs the standard Connectors into the current namespace",
		Long:  `This command will install the default Connectors into the current namespace`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			p.names = args
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
	f.StringVarP(&p.mavenRepo, "maven-repo", "m", "https://repo1.maven.org/maven2/", "the maven repository used to download the Connector releases")
	f.StringVarP(&p.namespace, "namespace", "n", "", "the namespace to query")
	f.StringVarP(&p.version, "version", "v", "latest", "the version of the connectors to install")
	f.BoolVar(&p.replace, "replace", false, "if enabled we will replace exising Connectors with installed version")
	f.BoolVar(&p.listConnectors, "list", false, "list all the available Connectors but don't install them")
	return cmd
}

func (p *installCmd) run() error {
	mavenRepo := p.mavenRepo
	version, err := versionForUrl(p.version, urlJoin(mavenRepo, connectorMetadataUrl))
	if err != nil {
		return err
	}
	uri := fmt.Sprintf(urlJoin(mavenRepo, connectorPackageUrlPrefix), version) + "kubernetes.yml"
	resp, err := http.Get(uri)
	if err != nil {
		return fmt.Errorf("Cannot load YAML package at %s got: %v", uri, err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Cannot load YAML from %s got: %v", uri, err)
	}
	list := v1.List{}
	err = yaml.Unmarshal(data, &list)
	if err != nil {
		return fmt.Errorf("Cannot parse YAML from %s got: %v", uri, err)
	}
	listOpts, err := funktion.CreateConnectorListOptions()
	if err != nil {
		return err
	}
	cms := p.kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return err
	}
	existingNames := map[string]bool{}
	for _, resource := range resources.Items {
		existingNames[resource.Name] = true
	}
	onlyNames := map[string]bool{}
	for _, onlyName := range p.names {
		onlyNames[onlyName] = true
	}

	if p.listConnectors {
		fmt.Printf("Version %s has Connectors:\n", version)
	}

	count := 0
	ignored := 0
	for _, item := range list.Items {
		cm, err := toConfigMap(&item)
		if err != nil {
			return err
		}
		name := cm.Name
		if p.listConnectors {
			fmt.Println(name)
		} else {
			if len(onlyNames) > 0 {
				if !onlyNames[name] {
					continue
				}
			}
			update := false
			operation := "create"
			if existingNames[name] {
				if p.replace {
					update = true
				} else {
					ignored++
					continue
				}
			}

			if update {
				operation = "update"
				_, err = cms.Update(cm)
			} else {
				_, err = cms.Create(cm)
			}
			if err != nil {
				return fmt.Errorf("Failed to %s Connector %s due to %v", operation, name, err)
			}
		}
		count++
	}

	if p.listConnectors {
		return nil
	}

	ignoreMessage := ""
	if !p.replace && ignored > 0 {
		ignoreMessage = fmt.Sprintf(". Ignored %d Connectors as they are already installed. (Please use `--replace` to force replacing them)", ignored)
	}

	fmt.Printf("Installed %d Connectors from version: %s%s\n", count, version, ignoreMessage)
	return nil
}

func toConfigMap(item *runtime.RawExtension) (*v1.ConfigMap, error) {
	obj := item.Object
	switch c := obj.(type) {
	case *v1.ConfigMap:
		return c, nil
	default:
		raw := item.Raw
		cm := v1.ConfigMap{}
		err := yaml.Unmarshal(raw, &cm)
		return &cm, err
	}
}

func versionForUrl(v string, metadataUrl string) (string, error) {
	resp, err := http.Get(metadataUrl)
	if err != nil {
		return "", fmt.Errorf("Cannot get version to deploy from url %s due to: %v", metadataUrl, err)
	}
	defer resp.Body.Close()
	// read xml http response
	xmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Cannot read version metadata from url %s due to: %v", metadataUrl, err)
	}

	type Metadata struct {
		Release  string   `xml:"versioning>release"`
		Versions []string `xml:"versioning>versions>version"`
	}

	var m Metadata
	err = xml.Unmarshal(xmlData, &m)
	if err != nil {
		return "", fmt.Errorf("Cannot parse version XML from url %s due to: %v", metadataUrl, err)
	}

	if v == "latest" {
		return m.Release, nil
	}

	for _, version := range m.Versions {
		if v == version {
			return version, nil
		}
	}
	return "", fmt.Errorf("Unknown version %s from URL %s when had valid version %v", v, metadataUrl, append(m.Versions, "latest"))
}



// urlJoin joins the given URL paths so that there is a / separating them but not a double //
func urlJoin(repo string, path string) string {
	return strings.TrimSuffix(repo, "/") + "/" + strings.TrimPrefix(path, "/")
}

