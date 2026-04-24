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

package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	"github.com/centos-automotive-suite/bob/internal/buildapi"
	"github.com/centos-automotive-suite/bob/internal/controller"
	"github.com/centos-automotive-suite/bob/internal/tekton"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(buildv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var apiAddr string
	var cliDir string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&apiAddr, "api-bind-address", ":8082", "The address the Build API binds to.")
	var artifactsDir string
	var pipelineAPIHost string
	var pipelineAPIPort string
	var maxUploadBytes int64
	var maxUploadFiles int
	var maxFileBytes int64
	flag.StringVar(&cliDir, "cli-dir", "/cli", "Directory containing bob CLI binaries for download.")
	flag.StringVar(&artifactsDir, "artifacts-dir", "/data/artifacts", "Directory for storing build artifacts.")
	flag.StringVar(&pipelineAPIHost, "pipeline-api-host", "", "Build API host for generated pipeline tasks (default: bob-api.<operator-namespace>.svc).")
	flag.StringVar(&pipelineAPIPort, "pipeline-api-port", "", "Build API port for generated pipeline tasks (default: 8082).")
	flag.Int64Var(&maxUploadBytes, "max-upload-bytes", buildapi.DefaultMaxUploadBytes, "Maximum total upload size in bytes.")
	flag.IntVar(&maxUploadFiles, "max-upload-files", buildapi.DefaultMaxUploadFiles, "Maximum number of files per upload.")
	flag.Int64Var(&maxFileBytes, "max-file-bytes", buildapi.DefaultMaxFileBytes, "Maximum size of a single uploaded file in bytes.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "bob.builder.sdv.cloud.redhat.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if pipelineAPIHost == "" {
		pipelineAPIHost = "bob-api." + operatorNamespace() + ".svc"
	}
	pipelineCfg := tekton.PipelineConfig{
		APIHost: pipelineAPIHost,
		APIPort: pipelineAPIPort,
	}
	if err := (&controller.BuildJobReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		PipelineConfig: pipelineCfg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BuildJob")
		os.Exit(1)
	}

	if err := (&controller.ToolchainReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Toolchain")
		os.Exit(1)
	}

	apiServer := buildapi.NewServer(mgr.GetClient(), apiAddr, cliDir, artifactsDir, cfg)
	apiServer.MaxUploadBytes = maxUploadBytes
	apiServer.MaxUploadFiles = maxUploadFiles
	apiServer.MaxFileBytes = maxFileBytes
	if err := mgr.Add(apiServer); err != nil {
		setupLog.Error(err, "unable to add API server")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager", "pipelineAPIHost", pipelineAPIHost)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func operatorNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return string(data)
	}
	return "bob-system"
}
