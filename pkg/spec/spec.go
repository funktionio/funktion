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

package spec

import (
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
)

const (
	EndpointKind   = "endpoint"
	FunctionKind   = "function"
	SetBodyKind    = "setBody"
	SetHeadersKind = "setHeaders"
)

// Connector defines how to create a Deployment for a Flow
type Connector struct {
	unversioned.TypeMeta `json:",inline"`
	v1.ObjectMeta        `json:"metadata,omitempty"`
	Spec                 ConnectorSpec `json:"spec"`
}

// ConnectorList is a list of Funktion.
type ConnectorList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`

	Items []*Connector `json:"items"`
}

// ConnectorSpec holds specification parameters of a Flow deployment along with configuration metadata.
type ConnectorSpec struct {
	DeploymentSpec *v1beta1.DeploymentSpec `json:"deploymentSpec"`

	// TODO lets add a JSON Schema for how to configure the endpoints?
}

// ComponentSpec holds the component metadata in a ConnectorSchema
type ComponentSpec struct {
	Kind        string `json:"kind"`
	Scheme      string `json:"scheme"`
	Syntax      string `json:"syntax"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Label       string `json:"label"`
	Deprecated  bool   `json:"deprecated"`
	Async       bool   `json:"async"`
	JavaType    string `json:"javaType"`
	GroupId     string `json:"groupId"`
	ArtifactId  string `json:"artifactId"`
	Version     string `json:"version"`
}

// PropertySpec contains the metadata for an individual property on a component or endpoint
type PropertySpec struct {
	Kind        string   `json:"kind"`
	Group       string   `json:"group"`
	Label       string   `json:"label"`
	Required    bool     `json:"required"`
	Type        string   `json:"type"`
	JavaType    string   `json:"javaType"`
	Enum        []string `json:"enum"`
	Deprecated  bool     `json:"deprecated"`
	Secret      bool     `json:"secret"`
	Description string   `json:"description"`
}

// ConnectorSchema holds the connector schema and metadata for the connector
type ConnectorSchema struct {
	Component           ComponentSpec           `json:"component"`
	ComponentProperties map[string]PropertySpec `json:"componentProperties"`
	Properties          map[string]PropertySpec `json:"properties"`
}

type FunkionConfig struct {
	Flows []FunktionFlow `json:"flows"`
}

type FunktionFlow struct {
	Name      string         `json:"name,omitempty"`
	Trace     bool           `json:"trace,omitempty"`
	LogResult bool           `json:"logResult,omitempty"`
	Steps     []FunktionStep `json:"steps"`
}

type FunktionStep struct {
	Kind    string            `json:"kind"`
	Name    string            `json:"name,omitempty"`
	URI     string            `json:"uri,omitempty"`
	Body    string            `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}
