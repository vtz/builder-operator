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

package buildapi

import buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"

type BuildJobSummary struct {
	Name         string      `json:"name"`
	Namespace    string      `json:"namespace"`
	Phase        string      `json:"phase"`
	Board        string      `json:"board,omitempty"`
	Platform     string      `json:"platform,omitempty"`
	Architecture string      `json:"architecture,omitempty"`
	Image        string      `json:"image,omitempty"`
	ArtifactURI  string      `json:"artifactURI,omitempty"`
	Stages       []StageInfo `json:"stages,omitempty"`
	PipelineRun  string      `json:"pipelineRun,omitempty"`
	Source       *SourceInfo `json:"source,omitempty"`
	Age          string      `json:"age,omitempty"`
}

type StageInfo struct {
	Name    string `json:"name"`
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

type SourceInfo struct {
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
	Revision string `json:"revision,omitempty"`
}

type BuildJobListResponse struct {
	Items []BuildJobSummary `json:"items"`
}

type CreateBuildJobRequest struct {
	Name string                     `json:"name"`
	Spec buildv1alpha1.BuildJobSpec `json:"spec"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
