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
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
)

const (
	// Connector

	// DeploymentYmlProperty data key for the deployment yaml file
	DeploymentYmlProperty = "deployment.yml"

	// SchemaYmlProperty data key for the schema (JSON schema as YAML) file
	SchemaYmlProperty = "schema.yml"

	// Flow

	// FunktionYmlProperty the data key for the funktion yaml file
	FunktionYmlProperty = "funktion.yml"

	// ApplicationPropertiesProperty is the data key for the spring boot application.properties file
	ApplicationPropertiesProperty = "application.properties"

	// ApplicationYmlProperty is the data key for the spring boot application.yml file
	ApplicationYmlProperty = "application.yml"

	// for Function

	// SourceProperty is the data key for the source code in a Function ConfigMap
	SourceProperty = "source"
	// DebugProperty is the data key for whether to enable debugging in a Function ConfigMap
	DebugProperty = "debug"
	// EnvVarsProperty represents a newline terminated list of NAME=VALUE expressions for environment variables
	EnvVarsProperty = "envVars"

	// ExposeLabel is the label key to expose services
	ExposeLabel = "expose"

	// for Runtime

	// DeploymentProperty is the data key for a Runtime's Deployment
	DeploymentProperty = "deployment"
	// DeploymentDebugProperty is the data key for a Runtime's Debug Deployment
	DeploymentDebugProperty = "deploymentDebug"
	// ServiceProperty is the data key for a Runtime's Service
	ServiceProperty = "service"
	// DebugPortProperty is the data key for a Runtime's debug port
	DebugPortProperty = "debugPort"

	// ConfigMapControllerAnnotation is the annotation for the configmapcontroller
	ConfigMapControllerAnnotation = "configmap.fabric8.io/update-on-change"

	// Deployment
	// NameLabel is the name label for Deployments
	NameLabel = "name"
)

func makeFlowDeployment(flow *v1.ConfigMap, connector *v1.ConfigMap, old *v1beta1.Deployment) (*v1beta1.Deployment, error) {
	deployYaml := connector.Data[DeploymentYmlProperty]
	if len(deployYaml) == 0 {
		return nil, fmt.Errorf("No property `%s` on the Flow ConfigMap %s", DeploymentYmlProperty, flow.Name)
	}

	deployment := v1beta1.Deployment{}
	err := yaml.Unmarshal([]byte(deployYaml), &deployment)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse Deployment YAML from property `%s` on the Flow ConfigMap %s. Error: %s", DeploymentYmlProperty, flow.Name, err)
	}

	name := flow.Name
	deployment.Name = name
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}
	if deployment.Labels == nil {
		deployment.Labels = make(map[string]string)
	}

	// lets copy across any old missing dependencies
	if old != nil {
		if old.Annotations != nil {
			for k, v := range old.Annotations {
				if len(deployment.Annotations[k]) == 0 {
					deployment.Annotations[k] = v
				}
			}
		}
	}
	if flow.Labels != nil {
		for k, v := range flow.Labels {
			if len(deployment.Labels[k]) == 0 {
				deployment.Labels[k] = v
			}
		}
	}
	if len(deployment.Annotations[ConfigMapControllerAnnotation]) == 0 {
		deployment.Annotations[ConfigMapControllerAnnotation] = flow.Name
	}

	volumeName := "config"
	items := []v1.KeyToPath{}

	// lets mount any files from the ConfigMap
	if len(flow.Data[FunktionYmlProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key:  FunktionYmlProperty,
			Path: "funktion.yml",
		})
	}
	if len(flow.Data[ApplicationPropertiesProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key:  ApplicationPropertiesProperty,
			Path: "application.properties",
		})
	}
	if len(flow.Data[ApplicationYmlProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key:  ApplicationYmlProperty,
			Path: "application.yml",
		})
	}
	if len(items) > 0 {
		podSpec := &deployment.Spec.Template.Spec
		podSpec.Volumes = append(podSpec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: flow.Name,
					},
					Items: items,
				},
			},
		})
		for i, container := range podSpec.Containers {
			podSpec.Containers[i].VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				Name:      volumeName,
				MountPath: "/deployments/config",
				ReadOnly:  true,
			})
		}
	}
	if len(deployment.Spec.Template.Spec.Containers[0].Name) == 0 {
		deployment.Spec.Template.Spec.Containers[0].Name = "connector"
	}
	setDeploymentLabel(&deployment, NameLabel, name)
	return &deployment, nil
}

func makeFunctionDeployment(function *v1.ConfigMap, runtime *v1.ConfigMap, old *v1beta1.Deployment) (*v1beta1.Deployment, error) {
	deployYaml := runtime.Data[DeploymentProperty]
	debugFlag := function.Data[DebugProperty]
	if strings.ToLower(debugFlag) == "true" {
		deployYaml = runtime.Data[DeploymentDebugProperty]
		if len(deployYaml) == 0 {
			return nil, fmt.Errorf("No property `%s` on the Runtime ConfigMap %s", DeploymentDebugProperty, runtime.Name)
		}
	}
	if len(deployYaml) == 0 {
		return nil, fmt.Errorf("No property `%s` on the Runtime ConfigMap %s", DeploymentProperty, runtime.Name)
	}

	deployment := v1beta1.Deployment{}
	err := yaml.Unmarshal([]byte(deployYaml), &deployment)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse Deployment YAML from property `%s` on the Runtime ConfigMap %s. Error: %s", DeploymentYmlProperty, runtime.Name, err)
	}

	name := function.Name
	deployment.Name = name
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}
	if deployment.Labels == nil {
		deployment.Labels = make(map[string]string)
	}

	// lets copy across any old missing dependencies
	if old != nil {
		if old.Annotations != nil {
			for k, v := range old.Annotations {
				if len(deployment.Annotations[k]) == 0 {
					deployment.Annotations[k] = v
				}
			}
		}
	}
	if function.Labels != nil {
		for k, v := range function.Labels {
			if len(deployment.Labels[k]) == 0 {
				deployment.Labels[k] = v
			}
		}
	}
	if len(deployment.Annotations[ConfigMapControllerAnnotation]) == 0 {
		deployment.Annotations[ConfigMapControllerAnnotation] = function.Name
	}

	if len(function.Data[SourceProperty]) == 0 {
		return nil, fmt.Errorf("No property `%s` on the Function ConfigMap %s", SourceProperty, function.Name)
	}

	volumeName := "config"
	items := []v1.KeyToPath{
		v1.KeyToPath{
			Key:  SourceProperty,
			Path: "source.js",
		},
	}

	foundVolume := false
	podSpec := &deployment.Spec.Template.Spec
	for i, volume := range podSpec.Volumes {
		if volume.Name == "source" && volume.ConfigMap != nil {
			podSpec.Volumes[i].ConfigMap.Name = function.Name
			foundVolume = true
		}
	}
	if !foundVolume {
		podSpec.Volumes = append(podSpec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: function.Name,
					},
					Items: items,
				},
			},
		})
	}

	envVars := parseEnvVars(function.Data[EnvVarsProperty])

	mountPath := runtime.Data[SourceMountPathProperty]
	if len(mountPath) == 0 {
		mountPath = "/funktion"
	}
	for i, container := range podSpec.Containers {
		foundVolumeMount := false
		for _, volumeMount := range container.VolumeMounts {
			if volumeMount.Name == "source" {
				foundVolumeMount = true
			}
		}
		if !foundVolumeMount {
			podSpec.Containers[i].VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				Name:      "source",
				MountPath: mountPath,
				ReadOnly:  true,
			})
		}
		if len(envVars) > 0 {
			applyEnvVars(&podSpec.Containers[i].Env, &envVars)
		}
	}
	if len(deployment.Spec.Template.Spec.Containers[0].Name) == 0 {
		deployment.Spec.Template.Spec.Containers[0].Name = "function"
	}
	setDeploymentLabel(&deployment, NameLabel, name)
	return &deployment, nil
}

func parseEnvVars(text string) []v1.EnvVar {
	answer := []v1.EnvVar{}
	if len(text) > 0 {
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			l := strings.TrimSpace(line)
			pair := strings.SplitN(l, "=", 2)
			if len(pair) != 2 {
				fmt.Printf("Ignoring bad environment variable pair. Expecting `NAME=VALUE` but got: %s\n", l)
				continue
			}
			answer = append(answer, v1.EnvVar{
				Name: pair[0],
				Value: pair[1],
			})
		}
		return answer
	}
	return answer
}

func applyEnvVars(envVar *[]v1.EnvVar, overrides *[]v1.EnvVar) {
	if overrides == nil {
		return

	}
	if *envVar == nil {
		*envVar = []v1.EnvVar{}
	}
	for _, o := range *overrides {
		found := false
		for _, v := range *envVar {
			if v.Name == o.Name {
				v.Value = o.Value
				v.ValueFrom = nil
				found = true
			}
		}
		if !found {
			*envVar = append(*envVar, o)
		}
	}
}

func setDeploymentLabel(deployment *v1beta1.Deployment, key string, value string) {
	deployment.Labels[key] = value
	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = &v1beta1.LabelSelector{}
	}
	if deployment.Spec.Selector.MatchLabels == nil {
		deployment.Spec.Selector.MatchLabels = map[string]string{}
	}
	deployment.Spec.Selector.MatchLabels[key] = value
	if deployment.Spec.Template.ObjectMeta.Labels == nil {
		deployment.Spec.Template.ObjectMeta.Labels = map[string]string{}
	}
	deployment.Spec.Template.ObjectMeta.Labels[key] = value
}
func makeFunctionService(function *v1.ConfigMap, runtime *v1.ConfigMap, old *v1.Service, deployment *v1beta1.Deployment) (*v1.Service, error) {
	yamlText := runtime.Data[ServiceProperty]
	if len(yamlText) == 0 {
		return nil, fmt.Errorf("No property `%s` on the Runtime ConfigMap %s", ServiceProperty, runtime.Name)
	}

	svc := &v1.Service{}
	err := yaml.Unmarshal([]byte(yamlText), &svc)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse Service YAML from property `%s` on the Runtime ConfigMap %s. Error: %s", DeploymentYmlProperty, runtime.Name, err)
	}

	svc.Name = function.Name
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}
	if svc.Labels == nil {
		svc.Labels = make(map[string]string)
	}

	svc.Spec.Selector = deployment.Spec.Selector.MatchLabels

	// lets copy across any old missing dependencies
	if old != nil {
		if old.Annotations != nil {
			for k, v := range old.Annotations {
				if len(svc.Annotations[k]) == 0 {
					svc.Annotations[k] = v
				}
			}
		}
	}
	if function.Labels != nil {
		for k, v := range function.Labels {
			if len(svc.Labels[k]) == 0 {
				svc.Labels[k] = v
			}
		}
	}
	if len(svc.Labels[ExposeLabel]) == 0 {
		svc.Labels[ExposeLabel] = "true"
	}
	return svc, nil
}
