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

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type BuildConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *BuildConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var bc buildv1alpha1.BuildConfig
	if err := r.Get(ctx, req.NamespacedName, &bc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	bc.Status.ObservedGeneration = bc.Generation

	ready := true
	reason := "ConfigurationValid"
	message := "BuildConfig defaults are active"

	if bc.Spec.Defaults.Timeout == "" {
		bc.Spec.Defaults.Timeout = "30m"
	}

	if bc.Spec.Compliance.Enabled {
		if bc.Spec.Compliance.SBOMFormat == "" {
			ready = false
			reason = "ComplianceMisconfigured"
			message = "compliance.enabled is true but sbomFormat is not set"
		}
	}

	status := metav1.ConditionTrue
	if !ready {
		status = metav1.ConditionFalse
	}

	bc.Status.Conditions = mergeCondition(bc.Status.Conditions,
		buildv1alpha1.NewCondition("Ready", status, reason, message, bc.Generation))

	if err := r.Status().Update(ctx, &bc); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating BuildConfig status: %w", err)
	}

	logger.Info("reconciled BuildConfig", "name", bc.Name, "ready", ready)
	return ctrl.Result{}, nil
}

func (r *BuildConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildv1alpha1.BuildConfig{}).
		Complete(r)
}
