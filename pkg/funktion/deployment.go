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
)

const (
	ImageProperty = "image"
	FunktionYmlProperty = "funktion.yml"
	ApplicationPropertiesProperty = "application.properties"
	ApplicationYmlProperty = "application.yml"
)

func makeDeployment(cm v1.ConfigMap, old *v1beta1.Deployment) (*v1beta1.Deployment, error) {
	spec, err := makeDeploymentSpec(cm)
	if err != nil {
		return nil, err
	}
	deployment := &v1beta1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name: cm.Name,
		},
		Spec: *spec,
	}
	if old != nil {
		deployment.Annotations = old.Annotations
	}
	return deployment, nil
}

func makeDeploymentSpec(cm v1.ConfigMap) (*v1beta1.DeploymentSpec, error) {
	var replicas int32 = 1

	image := cm.Data[ImageProperty]
	if len(image) == 0 {
		return nil, fmt.Errorf("No property `%s` on the ConfigMap %s", ImageProperty, cm.Name)
	}

	volumeName := "config"
	volumes := []v1.Volume{}
	volumeMounts := []v1.VolumeMount{}
	items := []v1.KeyToPath{}

	// lets mount any files from the ConfigMap
	if len(cm.Data[FunktionYmlProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key: FunktionYmlProperty,
			Path: "funktion.yml",
		})
	}
	if len(cm.Data[ApplicationPropertiesProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key: ApplicationPropertiesProperty,
			Path: "application.properties",
		})
	}
	if len(cm.Data[ApplicationYmlProperty]) > 0 {
		items = append(items, v1.KeyToPath{
			Key: ApplicationYmlProperty,
			Path: "application.yml",
		})
	}
	if len(items) > 0 {
		volumes = append(volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: cm.Name,
					},
					Items: items,
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name: volumeName,
			MountPath: "/deployments/config",
			ReadOnly: true,
		});
	}

	return &v1beta1.DeploymentSpec{
		Replicas: &replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Labels: cm.Labels,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "funktion",
						Image: image,
						Ports: []v1.ContainerPort{
							/*
							{
								Name:          "https",
								ContainerPort: 8443,
								Protocol:      v1.ProtocolTCP,
							},
							*/
						},
						VolumeMounts: volumeMounts,
					},
				},
				Volumes: volumes,
			},
		},
	}, nil
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