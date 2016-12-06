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

	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/util/intstr"
	"github.com/ghodss/yaml"
)

const (
	DeploymentYmlProperty = "deployment.yml"
	FunktionYmlProperty = "funktion.yml"
	ApplicationPropertiesProperty = "application.properties"
	ApplicationYmlProperty = "application.yml"
)

func makeDeployment(subscription *v1.ConfigMap, connector *v1.ConfigMap, old *v1beta1.Deployment) (*v1beta1.Deployment, error) {
	deployYaml := connector.Data[DeploymentYmlProperty]
	if len(deployYaml) == 0 {
		return nil, fmt.Errorf("No property `%s` on the Subscription ConfigMap %s", DeploymentYmlProperty, subscription.Name)
	}

	deployment := v1beta1.Deployment{}
	err := yaml.Unmarshal([]byte(deployYaml), &deployment)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse Deployment YAML from property `%s` on the Subscription ConfigMap %s. Error: %s", DeploymentYmlProperty, subscription.Name, err)
	}

	deployment.Name = subscription.Name
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
	if subscription.Labels != nil {
		for k, v := range subscription.Labels {
			if len(deployment.Labels[k]) == 0 {
				deployment.Labels[k] = v
			}
		}
	}

	volumeName := "config"
	items := []v1.KeyToPath{}

	// lets mount any files from the ConfigMap
	if len(subscription.Data[FunktionYmlProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key: FunktionYmlProperty,
			Path: "funktion.yml",
		})
	}
	if len(subscription.Data[ApplicationPropertiesProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key: ApplicationPropertiesProperty,
			Path: "application.properties",
		})
	}
	if len(subscription.Data[ApplicationYmlProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key: ApplicationYmlProperty,
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
						Name: subscription.Name,
					},
					Items: items,
				},
			},
		})
		for i, container := range podSpec.Containers {
			podSpec.Containers[i].VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				Name: volumeName,
				MountPath: "/deployments/config",
				ReadOnly: true,
			});
		}
	}
	if len(deployment.Spec.Template.Spec.Containers[0].Name) == 0 {
		deployment.Spec.Template.Spec.Containers[0].Name = "connector"
	}
	return &deployment, nil
}


func makeDeploymentService(cm v1.ConfigMap) *v1.Service {
	svc := &v1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name: "funktion",
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstr.FromString("https"),
				},
			},
			Selector: map[string]string{
				"app": "funktion",
			},
		},
	}
	return svc
}