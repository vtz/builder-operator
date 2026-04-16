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

type ToolQualification struct {
	// +optional
	// +kubebuilder:validation:Enum=TI1;TI2
	ToolImpact string `json:"toolImpact,omitempty"`
	// +optional
	// +kubebuilder:validation:Enum=TCL1;TCL2;TCL3;TCL4
	ToolConfidenceLevel string `json:"toolConfidenceLevel,omitempty"`
	// +optional
	EvidenceRef string `json:"evidenceRef,omitempty"`
	// +optional
	// +kubebuilder:validation:Enum=QM;A;B;C;D
	QualifiedForASIL string `json:"qualifiedForASIL,omitempty"`
}

type ToolchainBuildSpec struct {
	// Inline Dockerfile content. Mutually exclusive with ContextGit.
	// +optional
	Dockerfile string `json:"dockerfile,omitempty"`
	// Build from a Dockerfile in a git repository.
	// +optional
	ContextGit *GitSource `json:"contextGit,omitempty"`
	// Path to the Dockerfile within the git context (default: "Dockerfile").
	// +optional
	// +kubebuilder:default="Dockerfile"
	DockerfilePath string `json:"dockerfilePath,omitempty"`
}

type ToolchainCRSpec struct {
	// The image reference. If build is specified, this is the push destination.
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	// If specified, bob will build the toolchain image on-cluster.
	// +optional
	Build *ToolchainBuildSpec `json:"build,omitempty"`
	// +optional
	Platform string `json:"platform,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
	// +optional
	SupportedBoards []string `json:"supportedBoards,omitempty"`
	// +optional
	SupportedArchitectures []string `json:"supportedArchitectures,omitempty"`
	// +optional
	Qualification *ToolQualification `json:"qualification,omitempty"`
}

type ToolchainPhase string

const (
	ToolchainPhaseReady    ToolchainPhase = "Ready"
	ToolchainPhaseBuilding ToolchainPhase = "Building"
	ToolchainPhaseFailed   ToolchainPhase = "Failed"
)

type ToolchainCRStatus struct {
	// +optional
	Phase ToolchainPhase `json:"phase,omitempty"`
	// +optional
	ResolvedDigest string `json:"resolvedDigest,omitempty"`
	// +optional
	CurrentBuildRun string `json:"currentBuildRun,omitempty"`
	// +optional
	LastBuildTime string `json:"lastBuildTime,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=toolchains,scope=Namespaced,shortName=tc
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Platform",type=string,JSONPath=`.spec.platform`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type Toolchain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolchainCRSpec   `json:"spec,omitempty"`
	Status ToolchainCRStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ToolchainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Toolchain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Toolchain{}, &ToolchainList{})
}
