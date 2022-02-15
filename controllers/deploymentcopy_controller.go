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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	duplicationv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
)

var log = logf.Log.WithName("controller")

// DeploymentCopyReconciler reconciles a DeploymentCopy object
type DeploymentCopyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=duplication.k8s.wantedly.com,resources=deploymentcopies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=duplication.k8s.wantedly.com,resources=deploymentcopies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=duplication.k8s.wantedly.com,resources=deploymentcopies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DeploymentCopy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DeploymentCopyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the DeploymentCopy instance
	instance := &duplicationv1beta1.DeploymentCopy{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Spec.NameSuffix == "" {
		instance.Spec.NameSuffix = instance.Name
	}

	// TODO(munisystem): Set a status into the DeploymentCopy resource if the target deployment doesn't exist
	target, err := r.getDeployment(ctx, instance.Spec.TargetDeploymentName, instance.Namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	copied := target.DeepCopy()

	spec := copied.Spec
	if instance.Spec.Hostname != "" {
		spec.Template.Spec.Hostname = instance.Spec.Hostname
	}
	if instance.Spec.Replicas != 0 {
		spec.Replicas = &instance.Spec.Replicas
	}

	// Inject labels data into copied Deployment
	labels := map[string]string{}
	{
		for key, value := range copied.GetLabels() {
			labels[key] = value
		}
		for key, value := range instance.Spec.CustomLabels {
			labels[key] = value
			spec.Template.Labels[key] = value
			spec.Selector.MatchLabels[key] = value
		}
	}

	// Inject annotations data into copied Deployment
	annotations := map[string]string{}
	{
		for key, value := range copied.GetAnnotations() {
			annotations[key] = value
		}
		for key, value := range instance.Spec.CustomAnnotations {
			annotations[key] = value
		}
	}

	containers := make(map[string]duplicationv1beta1.Container, 0)
	for _, container := range instance.Spec.TargetContainers {
		containers[container.Name] = container
	}
	for i := range spec.Template.Spec.Containers {
		if container, ok := containers[spec.Template.Spec.Containers[i].Name]; ok {
			spec.Template.Spec.Containers[i].Image = container.Image
			spec.Template.Spec.Containers[i].Env = append(
				spec.Template.Spec.Containers[i].Env,
				container.Env...,
			)
		}
	}
	copiedDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", copied.ObjectMeta.Name, instance.Spec.NameSuffix),
			Namespace: instance.Namespace,
		},
	}

	log.Info("try to create or update copied Deployment", "namespace", copiedDeploy.Namespace, "name", copiedDeploy.Name)
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, copiedDeploy, func() error {
		copiedDeploy.Labels = labels
		copiedDeploy.Annotations = annotations
		copiedDeploy.Spec = spec

		// In order to support Update, set controller reference here
		return controllerutil.SetControllerReference(instance, copiedDeploy, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, errors.WithStack(err)
	}

	return reconcile.Result{}, nil
}

func (r *DeploymentCopyReconciler) getDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, found)
	return found, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentCopyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&duplicationv1beta1.DeploymentCopy{}).
		Complete(r)
}
