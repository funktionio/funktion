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

// Connector defines how to create a Deployment for a Subscription
type Connector struct {
	unversioned.TypeMeta `json:",inline"`
	v1.ObjectMeta        `json:"metadata,omitempty"`
	Spec ConnectorSpec `json:"spec"`
}

// ConnectorList is a list of Funktion.
type ConnectorList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`

	Items []*Connector `json:"items"`
}

// ConnectorSpec holds specification parameters of a Subscription deployment along with configuration metadata.
type ConnectorSpec struct {
	DeploymentSpec *v1beta1.DeploymentSpec `json:"deploymentSpec"`

	// TODO lets add a JSON Schema for how to configure the endpoints?
}
