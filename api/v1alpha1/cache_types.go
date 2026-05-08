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

type CacheType string

const (
	CacheTypeCCache  CacheType = "ccache"
	CacheTypeWest    CacheType = "west"
	CacheTypePip     CacheType = "pip"
	CacheTypeGo      CacheType = "go"
	CacheTypeGeneric CacheType = "generic"
)

type CacheCRSpec struct {
	// +kubebuilder:validation:Enum=ccache;west;pip;go;generic
	Type CacheType `json:"type"`
	// +kubebuilder:default="5Gi"
	StorageSize string `json:"storageSize,omitempty"`
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
	// +optional
	MaxAge string `json:"maxAge,omitempty"`
	// +kubebuilder:default=true
	Shared bool `json:"shared,omitempty"`
}

type CacheCRStatus struct {
	// +optional
	PVCName string `json:"pvcName,omitempty"`
	// +optional
	Phase string `json:"phase,omitempty"`
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=caches,scope=Namespaced
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.spec.storageSize`
// +kubebuilder:printcolumn:name="Shared",type=boolean,JSONPath=`.spec.shared`
// +kubebuilder:printcolumn:name="PVC",type=string,JSONPath=`.status.pvcName`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type Cache struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CacheCRSpec   `json:"spec,omitempty"`
	Status CacheCRStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type CacheList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cache `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cache{}, &CacheList{})
}
