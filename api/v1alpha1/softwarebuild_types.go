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

type SourceType string
type DestinationType string
type SoftwareBuildPhase string

const (
	SourceTypeGit      SourceType = "git"
	SourceTypePVC      SourceType = "pvc"
	SourceTypeHostPath SourceType = "hostPath"

	DestinationTypeSharedFolder DestinationType = "sharedFolder"
	DestinationTypeRegistry     DestinationType = "registry"
	DestinationTypeArtifactory  DestinationType = "artifactory"
	DestinationTypeQuay         DestinationType = "quay"

	PhasePending   SoftwareBuildPhase = "Pending"
	PhaseRunning   SoftwareBuildPhase = "Running"
	PhaseSucceeded SoftwareBuildPhase = "Succeeded"
	PhaseFailed    SoftwareBuildPhase = "Failed"
)

type SecretReference struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	Key string `json:"key,omitempty"`
}

type RuntimeSpec struct {
	// +kubebuilder:default=ubuntu:24.04
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type GitSource struct {
	// +kubebuilder:validation:Pattern=`^https?://|^git@`
	URL string `json:"url"`
	// +kubebuilder:default=main
	Revision string `json:"revision,omitempty"`
	// +optional
	CredentialsSecretRef *SecretReference `json:"credentialsSecretRef,omitempty"`
}

type PVCSource struct {
	// +kubebuilder:validation:MinLength=1
	ClaimName string `json:"claimName"`
	// +kubebuilder:default=/
	Path string `json:"path,omitempty"`
}

type HostPathSource struct {
	// +kubebuilder:validation:MinLength=1
	Path string `json:"path"`
}

type SourceSpec struct {
	// +kubebuilder:validation:Enum=git;pvc;hostPath
	Type SourceType `json:"type"`
	// +optional
	Git *GitSource `json:"git,omitempty"`
	// +optional
	PVC *PVCSource `json:"pvc,omitempty"`
	// +optional
	HostPath *HostPathSource `json:"hostPath,omitempty"`
}

type StageSpec struct {
	// +kubebuilder:validation:MinLength=1
	Command string `json:"command"`
	// +optional
	Image string `json:"image,omitempty"`
}

type PipelineStages struct {
	Fetch StageSpec `json:"fetch"`
	Prebuild StageSpec `json:"prebuild"`
	Build StageSpec `json:"build"`
	Postbuild StageSpec `json:"postbuild"`
	Deploy StageSpec `json:"deploy"`
}

type DestinationSpec struct {
	// +kubebuilder:validation:Enum=sharedFolder;registry;artifactory;quay
	Type DestinationType `json:"type"`
	// +optional
	Path string `json:"path,omitempty"`
	// +optional
	Repository string `json:"repository,omitempty"`
	// +optional
	CredentialsSecretRef *SecretReference `json:"credentialsSecretRef,omitempty"`
}

type SoftwareBuildSpec struct {
	// +optional
	Runtime RuntimeSpec `json:"runtime,omitempty"`
	Source SourceSpec `json:"source"`
	Stages PipelineStages `json:"stages"`
	Destination DestinationSpec `json:"destination"`
	// +optional
	TimeoutSeconds int64 `json:"timeoutSeconds,omitempty"`
}

type StageStatus struct {
	Name string `json:"name,omitempty"`
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// +optional
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
	// +optional
	State string `json:"state,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

type SoftwareBuildStatus struct {
	// +optional
	Phase SoftwareBuildPhase `json:"phase,omitempty"`
	// +optional
	CurrentPipelineRun string `json:"currentPipelineRun,omitempty"`
	// +optional
	ArtifactURI string `json:"artifactURI,omitempty"`
	// +optional
	FailureReason string `json:"failureReason,omitempty"`
	// +optional
	Stages []StageStatus `json:"stages,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=softwarebuilds,scope=Namespaced,shortName=sb
type SoftwareBuild struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SoftwareBuildSpec   `json:"spec,omitempty"`
	Status SoftwareBuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type SoftwareBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SoftwareBuild `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SoftwareBuild{}, &SoftwareBuildList{})
}
