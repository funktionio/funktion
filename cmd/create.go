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
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/funktionio/funktion/pkg/funktion"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/client-go/1.5/pkg/api/v1"
)

const (
	flowExtension = ".flow.yml"
)

type createFunctionCmd struct {
	createCmdCommon

	name          string
	runtime       string
	source        string
	file          string
	watch         bool
	debug         bool
	apply         bool
	functionsOnly bool

	envVars []string

	configMaps map[string]*v1.ConfigMap
}

func init() {
	RootCmd.AddCommand(newApplyCmd())
	RootCmd.AddCommand(newCreateCmd())
}

func newApplyCmd() *cobra.Command {
	p := &createFunctionCmd{
		apply: true,
	}
	cmd := &cobra.Command{
		Use:   "apply -f FILENAME",
		Short: "applies one or more resources from a file, directory or URL",
		Long:  `This command will apply (create or update) one or more resources from a file, directory or URL`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			err := createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
			if err != nil {
				handleError(err)
				return
			}
			handleError(p.createFromFile())
		},
	}

	cmd.AddCommand(newCreateFunctionCmd())
	cmd.AddCommand(newCreateFlowCmd())

	f := cmd.Flags()
	f.StringVarP(&p.file, "file", "f", "", "the file name that contains the source code for the function to create")
	p.setupCommonFlags(f)
	return cmd
}

func newCreateCmd() *cobra.Command {
	p := &createFunctionCmd{}
	cmd := &cobra.Command{
		Use:   "create -f FILENAME",
		Short: "creates one or more resources from the command line, a file, directory or URL",
		Long:  `This command will create one or more resources from the command one, a file, directory or URL`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			err := createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
			if err != nil {
				handleError(err)
				return
			}
			handleError(p.createFromFile())
		},
	}

	cmd.AddCommand(newCreateFunctionCmd())
	cmd.AddCommand(newCreateFlowCmd())

	f := cmd.Flags()
	f.StringVarP(&p.file, "file", "f", "", "the file name that contains the source code for the function to create")
	p.setupCommonFlags(f)
	return cmd
}

func newCreateFunctionCmd() *cobra.Command {
	p := &createFunctionCmd{}
	cmd := &cobra.Command{
		Use:   "fn [flags]",
		Short: "creates a new function",
		Long:  `This command will create a new function resource`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			err := createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
			if err != nil {
				handleError(err)
				return
			}
			handleError(p.createFunctionFromCLI())
		},
	}
	f := cmd.Flags()
	f.StringVarP(&p.name, "name", "n", "", "the name of the function to create")
	f.StringVarP(&p.source, "source", "s", "", "the source code of the function to create")
	f.StringVarP(&p.runtime, "runtime", "r", "nodejs", "the runtime to use. e.g. 'nodejs'")
	p.setupCommonFlags(f)
	return cmd
}

func (p *createFunctionCmd) createFromFile() error {
	file := p.file
	var err error
	if len(file) == 0 {
		return fmt.Errorf("No file argument specified!")
	}
	var matches []string
	if isExistingDir(file) {
		files, err := ioutil.ReadDir(file)
		if err != nil {
			return err
		}
		matches = []string{}
		for _, fi := range files {
			if !fi.IsDir() {
				matches = append(matches, filepath.Join(file, fi.Name()))
			}
		}
	} else {
		matches, err = filepath.Glob(file)
		if err != nil {
			return fmt.Errorf("Could not parse pattern %s due to %v", file, err)
		} else if len(matches) == 0 {
			fmt.Printf("No files exist matching the name: %s\n", file)
			fmt.Println("Please specify a file name that exists or specify the directory containing functions")
			return fmt.Errorf("No suitable source file: %s", file)
		}
	}
	for _, file := range matches {
		err = p.applyFile(file)
		if err != nil {
			return err
		}
	}
	if err == nil && p.watch {
		p.watchFiles()
	}
	return err
}

func (p *createFunctionCmd) setupCommonFlags(f *pflag.FlagSet) {
	f.StringArrayVarP(&p.envVars, "env", "e", []string{}, "pass one or more environment variables using the form NAME=VALUE")
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	f.StringVar(&p.namespace, "namespace", "", "the namespace to create the resource")
	f.BoolVarP(&p.watch, "watch", "w", false, "whether to keep watching the files for changes to the function source code")
	f.BoolVarP(&p.debug, "debug", "d", false, "enable debugging for the function?")
}

func (p *createFunctionCmd) createFunctionFromCLI() error {
	p.functionsOnly = true
	listOpts, err := funktion.CreateFunctionListOptions()
	if err != nil {
		return err
	}
	file := p.file
	if len(file) > 0 {
		return p.createFromFile()
	} else {
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

		name := nameFromFile(file, p.name)
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
			_, err = cms.Update(cm)
			message = "updated"
		} else {
			_, err = cms.Create(cm)
		}
		if err == nil {
			fmt.Printf("Function %s %s\n", name, message)
		}
	}
	if err == nil && p.watch {
		p.watchFiles()
	}
	return err
}

func (p *createFunctionCmd) watchFiles() {
	files := p.file
	if len(files) == 0 {
		return
	}
	fmt.Println("Watching files: ", files)
	fmt.Println("Please press Ctrl-C to terminate")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	matches, err := filepath.Glob(files)
	if err == nil {
		if len(matches) == 0 {
			// TODO could we watch for folder?
			log.Fatal("No files match pattern ", files)
			return
		}

	} else {
		matches = []string{files}
	}
	for _, file := range matches {
		err = watcher.Add(file)
		if err != nil {
			log.Fatal(err)
		}
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Rename == fsnotify.Rename {
				// if a file is renamed (e.g. IDE may do that) we no longer get any more events
				// so lets add the files again to be sure
				if isExistingFile(event.Name) {
					err = watcher.Add(event.Name)
					if err != nil {
						fmt.Printf("Failed to watch file %s due to %v\n", event.Name, err)
					}
				}
			}
			err = p.applyFile(event.Name)
			if err != nil {
				fmt.Printf("Failed to apply function file %s due to %v\n", event.Name, err)
			}

		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func isExistingFile(name string) bool {
	s, err := os.Stat(name)
	if err != nil {
		return false
	}
	return s.Mode().IsRegular()
}

func isExistingDir(name string) bool {
	s, err := os.Stat(name)
	if err != nil {
		return false
	}
	return s.Mode().IsDir()
}

func (p *createFunctionCmd) applyFile(fileName string) error {
	if !isExistingFile(fileName) {
		return nil
	}
	source, err := loadFileSource(fileName)
	if err != nil || len(source) == 0 {
		// ignore errors or blank source
		return nil
	}
	if !p.functionsOnly {
		if strings.HasSuffix(fileName, flowExtension) {
			return p.applyFlow(fileName, source)
		}
	}
	runtime, err := p.findRuntimeFromFileName(fileName)
	if err != nil {
		fmt.Printf("Failed to find runtime for file %s due to %v", fileName, err)
	}
	if len(runtime) == 0 {
		return nil
	}
	listOpts, err := funktion.CreateFunctionListOptions()
	if err != nil {
		return err
	}
	name := nameFromFile(fileName, "")
	if len(name) == 0 {
		return fmt.Errorf("Could not generate a function name!")
	}

	kubeclient := p.kubeclient
	cms := kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return err
	}
	var old *v1.ConfigMap = nil
	for _, resource := range resources.Items {
		if resource.Name == name {
			old = &resource
			break
		}
	}
	defaultLabels := map[string]string{}
	abs, err := filepath.Abs(fileName)
	if err == nil && len(abs) > 0 {
		folderName := convertToSafeLabelValue(filepath.Base(filepath.Dir(abs)))
		if len(folderName) > 0 {
			defaultLabels[funktion.ProjectLabel] = folderName
		}
	}
	cm, err := p.createFunctionFromSource(name, source, runtime, defaultLabels)
	if err != nil {
		return err
	}
	message := "created"
	if old != nil {
		oldSource := old.Data[funktion.SourceProperty]
		if source == oldSource && cm.Data[funktion.EnvVarsProperty] == old.Data[funktion.EnvVarsProperty] {
			// source not changed so lets not update!
			return nil
		}
		_, err = cms.Update(cm)
		message = "updated"
	} else {
		_, err = cms.Create(cm)
	}
	if err == nil {
		log.Println("Function", name, message)
	}
	return err
}

// findRuntimeFromFileName returns the runtime to use for the given file name
// or an empty string if the file does not map to a runtime function source file
func (p *createFunctionCmd) findRuntimeFromFileName(fileName string) (string, error) {
	// TODO we may want to use a cache and watch the runtimes to minimise API churn here on runtimes...
	listOpts, err := funktion.CreateRuntimeListOptions()
	if err != nil {
		return "", err
	}
	kubeclient := p.kubeclient
	cms := kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return "", err
	}
	ext := strings.TrimPrefix(filepath.Ext(fileName), ".")
	for _, resource := range resources.Items {
		data := resource.Data
		if data != nil {
			extensions := data[funktion.FileExtensionsProperty]
			if len(extensions) > 0 {
				values := strings.Split(extensions, ",")
				for _, value := range values {
					if ext == value {
						return resource.Name, nil
					}
				}
			}
		}
	}
	return "", nil
}

// returns the name of the resource to use given the fileName and the configured name
// if there is a configured name we will use that otherwise we will use the file name
// without the extension
func nameFromFile(fileName, configuredName string) string {
	if len(configuredName) != 0 {
		return convertToSafeResourceName(configuredName)
	}
	if len(fileName) == 0 {
		return ""
	}
	_, name := filepath.Split(fileName)
	ext := filepath.Ext(name)
	l := len(ext)
	if l > 0 {
		name = name[0 : len(name)-l]
	}
	return convertToSafeResourceName(name)
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
	source := p.source
	if len(source) == 0 {
		file := p.file
		if len(file) == 0 {
			return nil, fmt.Errorf("No function source code or file name supplied! You must specify either -s or -f flags")
		}
		var err error
		source, err = loadFileSource(file)
		if err != nil {
			return nil, err
		}
	}
	runtime := p.runtime
	if len(runtime) == 0 {
		return nil, fmt.Errorf("No runtime supplied! Please pass `-n nodejs` or some other valid runtime")
	}
	err := p.checkRuntimeExists(runtime)
	if err != nil {
		return nil, err
	}
	defaultLabels := map[string]string{}
	return p.createFunctionFromSource(name, source, runtime, defaultLabels)
}

func (p *createFunctionCmd) createFunctionFromSource(name, source, runtime string, extraLabels map[string]string) (*v1.ConfigMap, error) {
	labels := map[string]string{
		funktion.KindLabel:    funktion.FunctionKind,
		funktion.RuntimeLabel: runtime,
	}
	for k, v := range extraLabels {
		if labels[k] == "" {
			labels[k] = v
		}
	}
	data := map[string]string{
		funktion.SourceProperty: source,
	}
	if p.debug {
		data[funktion.DebugProperty] = "true"
	}
	if len(p.envVars) > 0 {
		data[funktion.EnvVarsProperty] = strings.Join(p.envVars, "\n")
	}
	cm := &v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Data: data,
	}
	return cm, nil
}

func loadFileSource(fileName string) (string, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	source := string(data)
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
