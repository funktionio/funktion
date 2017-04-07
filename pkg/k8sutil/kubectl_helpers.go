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

package k8sutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kardianos/osext"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/util/homedir"
)

// ResolveKubectlBinary resolves the binary to use such as 'kubectl' or 'oc'
func ResolveKubectlBinary(kubeclient *kubernetes.Clientset) (string, error) {
	isOpenshift := isOpenShiftCluster(kubeclient)
	kubeBinary := "kubectl"
	if isOpenshift {
		kubeBinary = "oc"
	}

	if runtime.GOOS == "windows" && !strings.HasSuffix(kubeBinary, ".exe") {
		kubeBinary += ".exe"
	}

	name, err := resolveBinaryLocation(kubeBinary)
	if err != nil {
		return "", err
	}
	if len(name) == 0 {
		return "", fmt.Errorf("Could not find binary %s on your PATH. Is it installed?", kubeBinary)
	}
	return name, nil
}

func isOpenShiftCluster(kubeclient *kubernetes.Clientset) bool {
	// The presence of "/oapi" on the API server is our hacky way of
	// determining if we're talking to OpenShift
	err := kubeclient.Core().GetRESTClient().Get().AbsPath("/oapi").Do().Error()
	if err != nil {
		return false
	}
	return true
}

// lets find the executable on the PATH or in the fabric8 directory
func resolveBinaryLocation(executable string) (string, error) {
	path, err := exec.LookPath(executable)
	if err != nil || fileNotExist(path) {
		home := os.Getenv("HOME")
		if home == "" {
			fmt.Printf("No $HOME environment variable found")
		}
		writeFileLocation := getFabric8BinLocation()

		// lets try in the fabric8 folder
		path = filepath.Join(writeFileLocation, executable)
		if fileNotExist(path) {
			path = executable
			// lets try in the folder where we found the gofabric8 executable
			folder, err := osext.ExecutableFolder()
			if err != nil {
				return "", fmt.Errorf("Failed to find executable folder: %v\n", err)
			} else {
				path = filepath.Join(folder, executable)
				if fileNotExist(path) {
					fmt.Printf("Could not find executable at %v\n", path)
					path = executable
				}
			}
		}
	}
	return path, nil
}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() {
		return nil
	}
	return os.ErrPermission
}

func fileNotExist(path string) bool {
	return findExecutable(path) != nil
}

func getFabric8BinLocation() string {
	home := homedir.HomeDir()
	if home == "" {
		fmt.Printf("No user home environment variable found for OS %s\n", runtime.GOOS)
	}
	return filepath.Join(home, ".fabric8", "bin")
}
