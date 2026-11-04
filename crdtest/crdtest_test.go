/*
Copyright 2026 Flant JSC

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

package crdtest_test

import (
	"context"
	"os"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/deckhouse/sds-common-lib/crdtest"
)

var k8sClient client.Client

// The directory also holds a doc-ru-*.yaml file, as a module's crds/ does; the
// environment only starts if it is skipped.
func TestMain(m *testing.M) {
	os.Exit(crdtest.Main(m, crdtest.Options{
		CRDPaths: []string{"testdata/crds"},
	}, func(env *crdtest.Env) { k8sClient = env.Client }))
}

func widget(name string, spec map[string]any) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{"spec": spec}}
	u.SetAPIVersion("example.deckhouse.io/v1alpha1")
	u.SetKind("Widget")
	u.SetName(name)
	return u
}

func requireInvalid(t *testing.T, err error, message string) {
	t.Helper()
	if !apierrors.IsInvalid(err) {
		t.Fatalf("want an Invalid error, got %v", err)
	}
	if !strings.Contains(err.Error(), message) {
		t.Fatalf("want %q in the error, got %v", message, err)
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		spec    map[string]any
		wantErr string
	}{
		{name: "valid", spec: map[string]any{"name": "a", "min": int64(1), "max": int64(2)}},
		{name: "schema pattern", spec: map[string]any{"name": "A"}, wantErr: "should match"},
		{name: "required field", spec: map[string]any{}, wantErr: "Required value"},
		{name: "CEL rule", spec: map[string]any{"name": "a", "min": int64(3), "max": int64(2)}, wantErr: "min must not exceed max"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			obj := widget("create-"+strings.ReplaceAll(tt.name, " ", "-"), tt.spec)
			err := k8sClient.Create(ctx, obj)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("create: %v", err)
				}
				t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), obj) })
				return
			}
			requireInvalid(t, err, tt.wantErr)
		})
	}
}

func TestUpdateTransitionRule(t *testing.T) {
	ctx := context.Background()

	obj := widget("transition", map[string]any{"name": "a"})
	if err := k8sClient.Create(ctx, obj); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), obj) })

	if err := unstructured.SetNestedField(obj.Object, "b", "spec", "name"); err != nil {
		t.Fatal(err)
	}
	requireInvalid(t, k8sClient.Update(ctx, obj), "name is immutable")
}

func TestStartWithoutCRDPaths(t *testing.T) {
	if _, err := crdtest.Start(crdtest.Options{}); err == nil {
		t.Fatal("want an error without CRDPaths")
	}
}
