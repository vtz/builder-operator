// Copyright 2026 Red Hat Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BuildDefaultsSpec struct {
	// +optional
	// +kubebuilder:default="30m"
	Timeout string `json:"timeout,omitempty"`
	// +optional
	// +kubebuilder:default="ubuntu:24.04"
	ToolchainImage string `json:"toolchainImage,omitempty"`
	// +optional
	CacheStorageClass string `json:"cacheStorageClass,omitempty"`
}

type ComplianceSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	// +kubebuilder:validation:Enum="spdx-json";"cyclonedx-json"
	SBOMFormat string `json:"sbomFormat,omitempty"`
	// +optional
	SigningEnabled bool `json:"signingEnabled,omitempty"`
}

type BuildAPISpec struct {
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// +kubebuilder:default=true
	RouteEnabled bool `json:"routeEnabled,omitempty"`
}

type BuildConfigSpec struct {
	// +optional
	Defaults BuildDefaultsSpec `json:"defaults,omitempty"`
	// +optional
	Compliance ComplianceSpec `json:"compliance,omitempty"`
	// +optional
	BuildAPI BuildAPISpec `json:"buildApi,omitempty"`
}

type BuildConfigStatus struct {
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=buildconfigs,scope=Cluster
type BuildConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildConfigSpec   `json:"spec,omitempty"`
	Status BuildConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type BuildConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildConfig{}, &BuildConfigList{})
}
