package testing

import (
	"bytes"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/pkg/errors"
	"github.com/stuart-warren/yamlfmt"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ddv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
)

type deploymentOption func(*appsv1.Deployment)
type deploymentCopyOption func(*ddv1beta1.DeploymentCopy)

func GenDeployment(name string, labels map[string]string, opts ...deploymentOption) *appsv1.Deployment {
	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "some-namespace",
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{},
			},
		},
	}

	for _, opt := range opts {
		opt(d)
	}
	return d
}

func AddContainer(name, image string) deploymentOption {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, v1.Container{Name: name, Image: image})
	}
}

func AddAnnotation(key, value string) deploymentOption {
	return func(d *appsv1.Deployment) {
		if d.ObjectMeta.Annotations == nil {
			d.ObjectMeta.Annotations = map[string]string{key: value}
			return
		}
		d.ObjectMeta.Annotations[key] = value
	}
}

func GenDeploymentCopy(name string, targetDeployment string, opts ...deploymentCopyOption) *ddv1beta1.DeploymentCopy {
	dc := &ddv1beta1.DeploymentCopy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "duplication.k8s.wantedly.com/v1beta1",
			Kind:       "DeploymentCopy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "some-namespace",
		},
		Spec: ddv1beta1.DeploymentCopySpec{
			TargetDeploymentName: targetDeployment,
		},
	}

	for _, opt := range opts {
		opt(dc)
	}
	return dc
}
func AddTargetContainer(name, image string) deploymentCopyOption {
	return func(dc *ddv1beta1.DeploymentCopy) {
		dc.Spec.TargetContainers = append(dc.Spec.TargetContainers, ddv1beta1.Container{
			Name:  name,
			Image: image,
			Env:   nil,
		})
	}
}
func AddCustomLabel(key, value string) deploymentCopyOption {
	return func(dc *ddv1beta1.DeploymentCopy) {
		if dc.Spec.CustomLabels == nil {
			dc.Spec.CustomLabels = map[string]string{key: value}
			return
		}
		dc.Spec.CustomLabels[key] = value
	}
}
func AddCustomAnnotation(key, value string) deploymentCopyOption {
	return func(dc *ddv1beta1.DeploymentCopy) {
		if dc.Spec.CustomAnnotations == nil {
			dc.Spec.CustomAnnotations = map[string]string{key: value}
			return
		}
		dc.Spec.CustomAnnotations[key] = value
	}
}

func SnapshotYaml(t *testing.T, objs ...interface{}) {
	t.Helper()

	manifests := make([]string, len(objs))

	for i, obj := range objs {

		// struct to map
		rs := make(map[string]interface{})
		{ // Marshal into json string to omit unused fields
			jsnStr, err := json.Marshal(obj)
			if err != nil {
				t.Fatal(err)
			}
			err = json.Unmarshal(jsnStr, &rs)
			if err != nil {
				t.Fatal(err)
			}
		}

		// map to formatted yaml
		var formatted string
		{
			d, err := yaml.Marshal(&rs)
			if err != nil {
				t.Fatal(err)
			}

			formatted, err = format(d)
			if err != nil {
				t.Fatal(err)
			}
		}
		manifests[i] = formatted
	}

	recorder := cupaloy.New(cupaloy.SnapshotFileExtension(".yaml"))
	recorder.SnapshotT(t, strings.Join(manifests, "\n"))
}

func format(content []byte) (string, error) {
	bs, err := yamlfmt.Format(bytes.NewReader(content))

	if err != nil {
		return "", errors.Wrap(err, "failed to format yaml")
	}
	return string(bs), nil
}
