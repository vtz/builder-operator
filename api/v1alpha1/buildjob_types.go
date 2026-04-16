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
type ArtifactDestinationType string
type BuildJobPhase string

const (
	SourceTypeGit SourceType = "git"
	SourceTypePVC SourceType = "pvc"

	ArtifactDestinationPVC ArtifactDestinationType = "pvc"
	ArtifactDestinationOCI ArtifactDestinationType = "oci"

	PhasePending   BuildJobPhase = "Pending"
	PhaseRunning   BuildJobPhase = "Running"
	PhaseSucceeded BuildJobPhase = "Succeeded"
	PhaseFailed    BuildJobPhase = "Failed"
)

type SecretReference struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	Key string `json:"key,omitempty"`
}

type ToolchainSpec struct {
	// +kubebuilder:default="ubuntu:24.04"
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

type SourceSpec struct {
	// +kubebuilder:validation:Enum=git;pvc
	Type SourceType `json:"type"`
	// +optional
	Git *GitSource `json:"git,omitempty"`
	// +optional
	PVC *PVCSource `json:"pvc,omitempty"`
}

type TargetSpec struct {
	// +optional
	Board string `json:"board,omitempty"`
	// +optional
	// +kubebuilder:validation:Enum=zephyr;openbsw;cmake;custom
	Platform string `json:"platform,omitempty"`
	// +optional
	// +kubebuilder:validation:Enum=arm;riscv;xtensa;x86;native
	Architecture string `json:"architecture,omitempty"`
	// +optional
	Variant string `json:"variant,omitempty"`
}

type StageSpec struct {
	// +kubebuilder:validation:MinLength=1
	Command string `json:"command"`
	// +optional
	Image string `json:"image,omitempty"`
}

type ArtifactSpec struct {
	// +kubebuilder:validation:Enum=pvc;oci
	// +kubebuilder:default=pvc
	Destination ArtifactDestinationType `json:"destination,omitempty"`
	// +optional
	Path string `json:"path,omitempty"`
}

type CacheMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
}

type BuildJobSpec struct {
	// +optional
	Toolchain ToolchainSpec `json:"toolchain,omitempty"`
	Source    SourceSpec    `json:"source"`
	// +optional
	Target TargetSpec `json:"target,omitempty"`

	// Stages is an ordered list of named build stages.
	// Each stage runs sequentially in the toolchain container.
	Stages []NamedStage `json:"stages"`

	// +optional
	Artifacts ArtifactSpec `json:"artifacts,omitempty"`
	// +optional
	Caches []CacheMount `json:"caches,omitempty"`
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

type NamedStage struct {
	// +kubebuilder:validation:MinLength=1
	Name      string `json:"name"`
	StageSpec `json:",inline"`
}

type StageStatus struct {
	Name string `json:"name,omitempty"`
	// +optional
	State string `json:"state,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

type BuildJobStatus struct {
	// +optional
	Phase BuildJobPhase `json:"phase,omitempty"`
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
	// +optional
	RunCount int64 `json:"runCount,omitempty"`
	// +optional
	LastRunAt string `json:"lastRunAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=buildjobs,scope=Namespaced,shortName=bj
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Board",type=string,JSONPath=`.spec.target.board`
// +kubebuilder:printcolumn:name="Platform",type=string,JSONPath=`.spec.target.platform`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type BuildJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildJobSpec   `json:"spec,omitempty"`
	Status BuildJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type BuildJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildJob{}, &BuildJobList{})
}
