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
	"os"
	"path/filepath"
	"strings"

	"github.com/funktionio/funktion/pkg/funktion"
	"github.com/funktionio/funktion/pkg/k8sutil"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"k8s.io/client-go/1.5/dynamic"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/runtime"
	"os/exec"
)

const (
	connectorMetadataUrl      = "io/fabric8/funktion/funktion-connectors/maven-metadata.xml"
	connectorPackageUrlPrefix = "io/fabric8/funktion/funktion-connectors/%[1]s/funktion-connectors-%[1]s-"
	runtimePackageUrlPrefix   = "io/fabric8/funktion/funktion-runtimes/%[1]s/funktion-runtimes-%[1]s-"

	operatorMetadataUrl      = "io/fabric8/platform/apps/funktion-operator/maven-metadata.xml"
	operatorPackageUrlPrefix = "io/fabric8/platform/apps/funktion-operator/%[1]s/funktion-operator-%[1]s-"

	platformMetadataUrl      = "io/fabric8/platform/packages/funktion-platform/maven-metadata.xml"
	platformPackageUrlPrefix = "io/fabric8/platform/packages/funktion-platform/%[1]s/funktion-platform-%[1]s-"
)

type installConnectorCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	namespace string
	names     []string
	version   string
	mavenRepo string
	replace   bool
	list      bool
	all       bool
}

type installRuntimeCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	namespace string
	names     []string
	version   string
	mavenRepo string
	replace   bool
	list      bool
	all       bool
}

type installPackageCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string
	dynamicClient  *dynamic.Client

	namespace string
	version   string
	mavenRepo string
	replace   bool

	packageMetadataUrl string
	packageUrlPrefix   string
}

func init() {
	RootCmd.AddCommand(newInstallCmd())
}

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [kind]",
		Short: "installs the standard Connectors into the current namespace",
		Long:  `This command will install the default Connectors into the current namespace`,
	}

	cmd.AddCommand(newInstallConnectorCmd())
	cmd.AddCommand(newInstallRuntimeCmd())
	cmd.AddCommand(newInstallOperatorCmd())
	cmd.AddCommand(newInstallPlatformCmd())
	return cmd
}

func newInstallConnectorCmd() *cobra.Command {
	p := &installConnectorCmd{}
	cmd := &cobra.Command{
		Use:   "connector [NAMES] [flags]",
		Short: "installs Connectors into the current namespace",
		Long:  `This command will install the standard Connectors into the current namespace`,
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
	f.BoolVarP(&p.list, "list", "l", false, "list all the available Connectors but don't install them")
	f.BoolVarP(&p.all, "all", "a", false, "Install all the connectors")
	return cmd
}

func newInstallRuntimeCmd() *cobra.Command {
	p := &installRuntimeCmd{}
	cmd := &cobra.Command{
		Use:   "runtime [NAMES] [flags]",
		Short: "installs the function Runtimes into the current namespace",
		Long:  `This command will install the standard Runtimes into the current namespace`,
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
	/*
		f.BoolVarP(&p.list, "list", "l", false, "list all the available Runtimes but don't install them")
		f.BoolVarP(&p.all, "all", "a", false, "Install all the runtimes")
	*/
	return cmd
}

func newInstallOperatorCmd() *cobra.Command {
	p := &installPackageCmd{
		packageMetadataUrl: operatorMetadataUrl,
		packageUrlPrefix:   operatorPackageUrlPrefix,
	}
	cmd := &cobra.Command{
		Use:   "operator [NAMES] [flags]",
		Short: "installs the Funktion Operator into the current namespace (when using fabric8)",
		Long: `This command will install the Funktion Operator into the current namespace

NOTE his command assumes you are already using the fabric8 developer platform - otherwise you should to install the 'platform' package`,
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
	p.configureFlags(cmd)
	return cmd
}

func newInstallPlatformCmd() *cobra.Command {
	p := &installPackageCmd{
		packageMetadataUrl: platformMetadataUrl,
		packageUrlPrefix:   platformPackageUrlPrefix,
	}
	cmd := &cobra.Command{
		Use:   "platform [NAMES] [flags]",
		Short: "installs the Funktion Platform into the current namespace (when not using fabric8)",
		Long: `This command will install the Funktion Platform into the current namespace

NOTE that if you are not already running the fabric8 developer platform - otherwise you only need to install the 'operator' package`,
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
	p.configureFlags(cmd)
	return cmd
}

func (p *installPackageCmd) configureFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	f.StringVarP(&p.mavenRepo, "maven-repo", "m", "https://repo1.maven.org/maven2/", "the maven repository used to download the Connector releases")
	f.StringVarP(&p.namespace, "namespace", "n", "funktion-system", "the namespace to query")
	f.StringVarP(&p.version, "version", "v", "latest", "the version of the connectors to install")
	f.BoolVar(&p.replace, "replace", false, "if enabled we will replace exising Connectors with installed version")

}
func (p *installConnectorCmd) run() error {
	mavenRepo := p.mavenRepo
	version, err := versionForUrl(p.version, urlJoin(mavenRepo, connectorMetadataUrl))
	if err != nil {
		return err
	}
	uri := fmt.Sprintf(urlJoin(mavenRepo, connectorPackageUrlPrefix), version) + "kubernetes.yml"
	return p.installConnectors(uri, version)
}

func (p *installConnectorCmd) installConnectors(uri string, version string) error {
	list, err := loadList(uri)
	if err != nil {
		return err
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

	if p.list {
		fmt.Printf("Version %s has Connectors:\n", version)
	} else if !p.all && len(onlyNames) == 0 {
		fmt.Printf("No connector names listed so none have been installed.\nPlease specify the connector names to install, use `--all-connectors` to install then all or use `--list-connectors` to list them\n")
		return nil
	}

	count := 0
	ignored := 0
	for _, item := range list.Items {
		cm, err := toConfigMap(&item)
		if err != nil {
			return err
		}
		name := cm.Name
		if p.list {
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

	if p.list {
		return nil
	}

	ignoreMessage := ""
	if !p.replace && ignored > 0 {
		ignoreMessage = fmt.Sprintf(". Ignored %d Connectors as they are already installed. (Please use `--replace` to force replacing them)", ignored)
	}

	fmt.Printf("Installed %d Connectors from version: %s%s\n", count, version, ignoreMessage)
	return nil
}

func (p *installRuntimeCmd) run() error {
	mavenRepo := p.mavenRepo
	version, err := versionForUrl(p.version, urlJoin(mavenRepo, connectorMetadataUrl))
	if err != nil {
		return err
	}
	uri := fmt.Sprintf(urlJoin(mavenRepo, runtimePackageUrlPrefix), version) + "kubernetes.yml"
	err = p.installRuntimes(uri, version)
	if err != nil {
		return err
	}
	return nil
}

func (p *installRuntimeCmd) installRuntimes(uri string, version string) error {
	list, err := loadList(uri)
	if err != nil {
		return err
	}
	listOpts, err := funktion.CreateRuntimeListOptions()
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
	count := 0
	ignored := 0
	for _, item := range list.Items {
		cm, err := toConfigMap(&item)
		if err != nil {
			return err
		}
		name := cm.Name
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
			return fmt.Errorf("Failed to %s Runtime %s due to %v", operation, name, err)
		}
		count++
	}

	ignoreMessage := ""
	if !p.replace && ignored > 0 {
		ignoreMessage = fmt.Sprintf(". Ignored %d Runtimes as they are already installed. (Please use `--replace` to force replacing them)", ignored)
	}

	fmt.Printf("Installed %d Runtimes from version: %s%s\n", count, version, ignoreMessage)
	return nil
}

func (p *installPackageCmd) run() error {
	mavenRepo := p.mavenRepo
	version, err := versionForUrl(p.version, urlJoin(mavenRepo, p.packageMetadataUrl))
	if err != nil {
		return err
	}
	uri := fmt.Sprintf(urlJoin(mavenRepo, p.packageUrlPrefix), version) + "kubernetes.yml"
	err = p.checkNamespaceExists()
	if err != nil {
		return err
	}
	err = p.installPackage(uri, version)
	if err != nil {
		return err
	}
	return nil
}

func (p *installPackageCmd) checkNamespaceExists() error {
	name := p.namespace
	namespaces := p.kubeclient.Namespaces()
	ns, err := namespaces.Get(name)
	if err != nil {
		ns = &v1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"system": "funktion",
				},
			},
		}
		_, err = namespaces.Create(ns)
		if err == nil {
			fmt.Printf("created Namespace %s\n", name)
		}
	}
	return err
}

func (p *installPackageCmd) installPackage(uri string, version string) error {
	binaryFile, err := k8sutil.ResolveKubectlBinary(p.kubeclient)
	if err != nil {
		return err
	}
	args := []string{"apply", "--namespace", p.namespace, "-f", uri}
	fmt.Printf("%s %s\n\n", filepath.Base(binaryFile), strings.Join(args, " "))
	cmd := exec.Command(binaryFile, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()

	/*
		TODO try use the dynamic client

			list, err := loadList(uri)
			if err != nil {
				return err
			}
			resources, err := p.kubeclient.ServerResources()
			if err != nil {
				return err
			}
			resourceMap := map[string]*unversioned.APIResource{}
			for _, ra := range resources {
				for _, r := range ra.APIResources {
					resourceMap[r.Kind] = &r
				}
			}
			client := p.dynamicClient
			ns := p.namespace
			count := 0
			m := meta.NewAccessor()
			for _, item := range list.Items {
				u := runtime.Unknown{Raw: item.Raw}
				kind := u.Kind
				resource := resourceMap[kind]
				if resource != nil {
					_, err := client.Resource(resource, ns).Create()
					if err != nil {
						return err
					}
					count++
				} else {
					fmt.Printf("Could not find resource for kind %s\n", kind)
				}
			}
			fmt.Printf("Installed %d resources from version: %s\n", count, version)
	*/
}

func loadList(uri string) (*v1.List, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("Cannot load YAML package at %s got: %v", uri, err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Cannot load YAML from %s got: %v", uri, err)
	}
	list := v1.List{}
	err = yaml.Unmarshal(data, &list)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse YAML from %s got: %v", uri, err)
	}
	return &list, nil
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
