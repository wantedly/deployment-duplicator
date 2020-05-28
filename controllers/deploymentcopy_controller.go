/*
Copyright 2020 Wantedly, Inc..

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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	duplicationv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
)

var log = logf.Log.WithName("controller")

// DeploymentCopyReconciler reconciles a DeploymentCopy object
type DeploymentCopyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=duplication.k8s.wantedly.com,resources=deploymentcopies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=duplication.k8s.wantedly.com,resources=deploymentcopies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

func (r *DeploymentCopyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// Fetch the DeploymentCopy instance
	instance := &duplicationv1beta1.DeploymentCopy{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Spec.NameSuffix == "" {
		instance.Spec.NameSuffix = instance.Name
	}

	// TODO(munisystem): Set a status into the DeploymentCopy resource if the target deployment doesn't exist
	target, err := r.getDeployment(instance.Spec.TargetDeploymentName, instance.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
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
			Name:        fmt.Sprintf("%s-%s", copied.ObjectMeta.Name, instance.Spec.NameSuffix),
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: spec,
	}

	if err := controllerutil.SetControllerReference(instance, copiedDeploy, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}
	_, err = r.getDeployment(copiedDeploy.Name, copiedDeploy.Namespace)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating copied Deployment", "namespace", copiedDeploy.Namespace, "name", copiedDeploy.Name)
		err = r.Create(context.TODO(), copiedDeploy)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *DeploymentCopyReconciler) getDeployment(name, namespace string) (*appsv1.Deployment, error) {
	found := &appsv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, found)
	return found, err
}

func (r *DeploymentCopyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&duplicationv1beta1.DeploymentCopy{}).
		Complete(r)
}
