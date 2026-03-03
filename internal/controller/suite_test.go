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

package controller

import (
	"context"
	"fmt"
	"os"
	"testing"

	buildv1alpha1 "github.com/example/builder-operator/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	testEnv    *envtest.Environment
	testClient client.Client
	cancelMgr  context.CancelFunc
)

// TestMain sets up envtest when KUBEBUILDER_ASSETS is provided, allowing
// integration tests to run alongside unit tests. Without that variable, only
// unit tests execute and integration tests are skipped.
func TestMain(m *testing.M) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	if os.Getenv("KUBEBUILDER_ASSETS") != "" {
		setupEnvtest()
	}

	code := m.Run()

	if testEnv != nil {
		if cancelMgr != nil {
			cancelMgr()
		}
		_ = testEnv.Stop()
	}

	os.Exit(code)
}

func setupEnvtest() {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		panic(fmt.Sprintf("failed to add client-go scheme: %v", err))
	}
	if err := buildv1alpha1.AddToScheme(scheme); err != nil {
		panic(fmt.Sprintf("failed to add buildv1alpha1 scheme: %v", err))
	}

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../config/crd/bases"},
		CRDs:                  []*apiextensionsv1.CustomResourceDefinition{minimalPipelineRunCRD()},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		panic(fmt.Sprintf("failed to start envtest: %v", err))
	}

	testClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		panic(fmt.Sprintf("failed to create test client: %v", err))
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create manager: %v", err))
	}

	if err := (&SoftwareBuildReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		panic(fmt.Sprintf("failed to set up reconciler: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelMgr = cancel
	go func() {
		if err := mgr.Start(ctx); err != nil && ctx.Err() == nil {
			panic(fmt.Sprintf("manager exited unexpectedly: %v", err))
		}
	}()
}

// minimalPipelineRunCRD returns a bare-bones CRD for tekton.dev/v1 PipelineRun
// so the API server accepts Create calls from the controller during tests.
func minimalPipelineRunCRD() *apiextensionsv1.CustomResourceDefinition {
	trueVal := true
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pipelineruns.tekton.dev",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "tekton.dev",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:   "PipelineRun",
				Plural: "pipelineruns",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type:                   "object",
							XPreserveUnknownFields: &trueVal,
						},
					},
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
				},
			},
		},
	}
}
