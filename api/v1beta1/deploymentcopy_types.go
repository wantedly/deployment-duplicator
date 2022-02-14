/*
Copyright 2022 Wantedly, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeploymentCopySpec defines the desired state of DeploymentCopy
type DeploymentCopySpec struct {
	// labels in `CustomLabels` and those of `TargetDeploymentName` will be merged.
	// When both have same keys, values in `Labels` will be applied
	// This will also used for `Spec.Template.Labels` and `Spec.Selector.MatchLabels` of copied Deployment
	CustomLabels map[string]string `json:"customLabels,omitempty"`

	// annotations in `CustomAnnotations` and those of `TargetDeploymentName` will be merged.
	// When both have same keys, values in `Labels` will be applied
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty"`

	// If non-zero, Replicas will be used for replicas for the copied deployment
	Replicas int32 `json:"replicas"`

	// name defined in `TargetDeploymentName` will be copied
	TargetDeploymentName string `json:"targetDeploymentName"`

	// (optional) if defined, the copied deployment will have the specified Hostname
	Hostname string `json:"hostname"`

	// (optional) if defined, the copied deployment will have suffix with this value.
	// When not defined, `.Matadata.Name` will be used
	NameSuffix string `json:"nameSuffix"`

	// name defined in `TargetDeploymentName` will be copied
	TargetContainers []Container `json:"targetContainers"`
}

// Container should be compatible with "k8s.io/api/apps/v1".Container, so that we can support more fields later on
type Container struct {
	Name  string      `json:"name"`
	Image string      `json:"image"`
	Env   []v1.EnvVar `json:"env"`
}

// DeploymentCopyStatus defines the observed state of DeploymentCopy
type DeploymentCopyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeploymentCopy is the Schema for the deploymentcopies API
type DeploymentCopy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentCopySpec   `json:"spec,omitempty"`
	Status DeploymentCopyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeploymentCopyList contains a list of DeploymentCopy
type DeploymentCopyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentCopy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentCopy{}, &DeploymentCopyList{})
}
