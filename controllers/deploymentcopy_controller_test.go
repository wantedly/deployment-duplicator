package controllers_test

import (
	"context"
	ddv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/wantedly/deployment-duplicator/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ut "github.com/wantedly/deployment-duplicator/controllers/testing"
	"testing"
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
			name:         "do nothing",
			explanation:  "",
			initialState: nil,
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
				Name:      "some-identifier",
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
