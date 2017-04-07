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

package funktion

import (
	"bytes"
	"fmt"

	"github.com/funktionio/funktion/pkg/spec"
	"github.com/ghodss/yaml"
	"strings"
	"unicode"
)

func LoadConnectorSchema(yamlData []byte) (*spec.ConnectorSchema, error) {
	schema := spec.ConnectorSchema{}
	err := yaml.Unmarshal(yamlData, &schema)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse schema YAML: %v", err)
	}
	return &schema, nil
}

// HumanizeString converts a camelCase text string into a textual label by capitalising
// and separating camelcase words with a space
func HumanizeString(text string) string {
	uncamel := UnCamelCaseString(text, " ")
	switch len(uncamel) {
	case 0:
		return ""
	case 1:
		return strings.ToUpper(uncamel)
	default:
		return strings.ToUpper(uncamel[0:1]) + uncamel[1:]
	}
}

// UnCamelCaseString converts a camelCase text string into a space separated string
func UnCamelCaseString(text string, separator string) string {
	var buffer bytes.Buffer
	lastUpper := false
	for i, c := range text {
		if unicode.IsUpper(c) {
			if !lastUpper && i > 0 {
				buffer.WriteString(separator)
			}
			lastUpper = true
		} else {
			lastUpper = false
		}
		buffer.WriteRune(c)
	}
	return buffer.String()
}

func ToSpringBootPropertyName(text string) string {
	return strings.ToLower(UnCamelCaseString(text, "-"))
}
