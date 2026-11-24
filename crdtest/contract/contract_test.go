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

package contract_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deckhouse/sds-common-lib/crdtest/contract"
)

var crds = []string{"testdata/crds"}

func messages(vs []contract.Violation) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.String())
	}
	return out
}

func TestConformingCRDs(t *testing.T) {
	contract.Require(t, crds,
		contract.Resource{Kind: "ExampleStorageConnection", Role: contract.Connection},
		contract.Resource{Kind: "ExampleStorageClass", Role: contract.Class},
	)
}

func TestViolations(t *testing.T) {
	vs, err := contract.Check(crds, contract.Resource{
		Kind:         "BrokenStorageConnection",
		Role:         contract.Connection,
		Sensitive:    []string{"spec.auth.chap", "spec.auth.missing"},
		NotSensitive: []string{},
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		"BrokenStorageConnection: R6: category d8-storage-connections is missing from spec.names.categories [d8-storage]",
		"BrokenStorageConnection: R6: short name bsconn is not in StorageShortNames; register it in sds-common-lib, where its uniqueness is checked",
		"BrokenStorageConnection: R6: short name lvg is already taken by LVMVolumeGroup",
		"BrokenStorageConnection v1alpha1: R6: column Backend is missing",
		"BrokenStorageConnection v1alpha1: R6: column Reason has priority 1, so kubectl get shows it only with -o wide",
		`BrokenStorageConnection v1alpha1: R6: column Reason reads .status.reason, not .status.conditions[?(@.type=="Ready")].reason`,
		"BrokenStorageConnection v1alpha1: R6: column Age has type string, not date",
		"BrokenStorageConnection v1alpha1: R6: column Phase has priority 0; it is shown only with -o wide (priority: 1)",
		"BrokenStorageConnection v1alpha1: R4: the status subresource is not enabled",
		"BrokenStorageConnection v1alpha1: R4: status.conditions is missing, so the Ready condition has nowhere to be",
		"BrokenStorageConnection v1alpha1: R3: spec.auth.chap holds credentials and is not marked x-kubernetes-sensitive-data: true",
		"BrokenStorageConnection v1alpha1: R3: spec.credentialsSecretName refers to a Secret; credentials are fields of the resource marked x-kubernetes-sensitive-data",
		"BrokenStorageConnection v1alpha1: R3: spec.password holds credentials and is not marked x-kubernetes-sensitive-data: true",
		"BrokenStorageConnection v1alpha1: R3: spec.auth.missing is listed in Sensitive and is not in the schema",
	}, messages(vs))
}

func TestNotSensitive(t *testing.T) {
	vs, err := contract.Check(crds, contract.Resource{
		Kind:         "BrokenStorageConnection",
		Role:         contract.Connection,
		NotSensitive: []string{"spec.password"},
	})
	require.NoError(t, err)
	assert.NotContains(t, messages(vs), "BrokenStorageConnection v1alpha1: R3: spec.password holds credentials and is not marked x-kubernetes-sensitive-data: true")
}

func TestSkipColumns(t *testing.T) {
	vs, err := contract.Check(crds, contract.Resource{
		Kind:        "BrokenStorageConnection",
		Role:        contract.Connection,
		SkipColumns: []string{contract.ColumnBackend},
	})
	require.NoError(t, err)
	assert.NotContains(t, messages(vs), "BrokenStorageConnection v1alpha1: R6: column Backend is missing")
}

func TestRoleDecidesTheCategoriesAndColumns(t *testing.T) {
	vs, err := contract.Check(crds, contract.Resource{Kind: "ExampleStorageClass", Role: contract.Connection})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"ExampleStorageClass: R6: category d8-storage-connections is missing from spec.names.categories [d8-storage d8-storage-classes]",
		"ExampleStorageClass v1alpha1: R6: column Backend is missing",
		"ExampleStorageClass v1alpha1: R6: column Address is missing",
	}, messages(vs))
}

func TestCheckErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		paths     []string
		resources []contract.Resource
	}{
		{name: "no resources", paths: crds},
		{name: "no paths", resources: []contract.Resource{{Kind: "ExampleStorageClass", Role: contract.Class}}},
		{name: "missing path", paths: []string{"testdata/none"}, resources: []contract.Resource{{Kind: "ExampleStorageClass", Role: contract.Class}}},
		{name: "unknown kind", paths: crds, resources: []contract.Resource{{Kind: "Nothing", Role: contract.Class}}},
		{name: "no kind", paths: crds, resources: []contract.Resource{{Role: contract.Class}}},
		{name: "unknown role", paths: crds, resources: []contract.Resource{{Kind: "ExampleStorageClass", Role: "pool"}}},
		{name: "column that cannot be skipped", paths: crds, resources: []contract.Resource{{Kind: "ExampleStorageClass", Role: contract.Class, SkipColumns: []string{contract.ColumnReady}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := contract.Check(tc.paths, tc.resources...)
			assert.Error(t, err)
		})
	}
}

func TestCheckReadsAFile(t *testing.T) {
	vs, err := contract.Check([]string{"testdata/crds/storage.example.io_examplestorageconnections.yaml"},
		contract.Resource{Kind: "ExampleStorageConnection", Role: contract.Connection})
	require.NoError(t, err)
	assert.Empty(t, vs)
}

type recorder struct {
	testing.TB
	errors []string
}

func (r *recorder) Helper() {}

func (r *recorder) Error(args ...any) { r.errors = append(r.errors, fmt.Sprint(args...)) }

func TestRequireReportsEachViolation(t *testing.T) {
	rec := &recorder{TB: t}
	contract.Require(rec, crds, contract.Resource{Kind: "ExampleStorageClass", Role: contract.Connection})
	assert.Len(t, rec.errors, 3)
}

func TestStorageShortNamesHaveOwners(t *testing.T) {
	for short, kind := range contract.StorageShortNames {
		assert.NotEmpty(t, short)
		assert.NotEmpty(t, kind, short)
	}
	for _, removed := range []string{"hsc", "hsconn"} {
		assert.NotContains(t, contract.StorageShortNames, removed)
	}
}
