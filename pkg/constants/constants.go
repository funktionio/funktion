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
package constants

import (
	"path/filepath"

	"k8s.io/client-go/1.5/pkg/util/homedir"
)

var FunktionPath = filepath.Join(homedir.HomeDir(), ".funktion")

// MakeFunktionPath is a utility to calculate a relative path to our directory.
func MakeFunktionPath(fileName ...string) string {
	args := []string{FunktionPath}
	args = append(args, fileName...)
	return filepath.Join(args...)
}

var ConfigFile = MakeFunktionPath("config", "config.json")
