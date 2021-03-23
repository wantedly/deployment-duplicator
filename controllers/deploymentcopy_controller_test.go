package controllers_test

import (
	"context"
	"testing"

	ddv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/wantedly/deployment-duplicator/controllers"
	ut "github.com/wantedly/deployment-duplicator/controllers/testing"
)

type testcase struct {
	name         string
	explanation  string
	initialState []runtime.Object
}

func TestDeploymentCopyReconciler(t *testing.T) {
	scheme := runtime.NewScheme()

	regs := []func(*runtime.Scheme) error{
		ddv1beta1.AddToScheme,
		clientgoscheme.AddToScheme,
	}

	for _, add := range regs {
		if err := add(scheme); err != nil {
			t.Fatal(err)
		}
	}

	testcases := []testcase{
		{
			name:         "no resources",
			explanation:  "do nothing",
			initialState: nil,
		},
		{
			name:        "only deployment copy",
			explanation: "do nothing because there's no deployment",
			initialState: []runtime.Object{
				ut.GenDeploymentCopy("some-deployment-copy", "some-deployment", ut.AddTargetContainer("some-container", "some-image-tag")),
			},
		},
		{
			name:        "one deployment and one deployment copy",
			explanation: "should make a copy",
			initialState: []runtime.Object{
				ut.GenDeployment("some-deployment", map[string]string{"app": "some-app", "role": "web"}, ut.AddContainer("some-container", "some-image-tag")),
				ut.GenDeploymentCopy("some-deployment-copy", "some-deployment", ut.AddTargetContainer("some-container", "another-image-tag")),
			},
		},
		{
			name:        "copied deployment exists",
			explanation: "should update it, therefore it can't update",
			initialState: []runtime.Object{
				ut.GenDeployment("some-deployment", map[string]string{"app": "some-app", "role": "web"}, ut.AddContainer("some-container", "some-image-tag")),
				ut.GenDeployment("some-deployment-some-deployment-copy", map[string]string{"app": "some-app", "role": "web"}, ut.AddContainer("some-container", "other-image-tag")),
				ut.GenDeploymentCopy("some-deployment-copy", "some-deployment", ut.AddTargetContainer("some-container", "another-image-tag")),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClientWithScheme(scheme, tc.initialState...)

			rec := controllers.DeploymentCopyReconciler{
				Client: client,
				Log:    ctrl.Log,
				Scheme: scheme,
			}

			ctx := context.Background()
			nn := types.NamespacedName{
				Namespace: "some-namespace",
				Name:      "some-deployment-copy",
			}
			req := ctrl.Request{NamespacedName: nn}
			if _, err := rec.Reconcile(req); err != nil {
				t.Fatalf("%+v", err)
			}

			lists := []runtime.Object{
				&ddv1beta1.DeploymentCopyList{},
				&appsv1.DeploymentList{},
			}

			for _, ls := range lists {
				if err := client.List(ctx, ls); err != nil {
					t.Fatalf("%+v", err)
				}
			}
			ifs := make([]interface{}, len(lists))
			for i, ls := range lists {
				ifs[i] = ls
			}
			ut.SnapshotYaml(t, ifs...)
		})
	}
}
