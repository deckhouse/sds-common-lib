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

// Package contract checks the CRDs of a storage module against the common
// resource contract of the CSI modules: categories, short names and printer
// columns (R6), the Ready condition in the status (R4) and credentials held
// in sensitive fields of the resource rather than in Secrets (R3).
//
// It reads the manifests and starts nothing, so a module calls it from any
// unit test:
//
//	func TestCRDContract(t *testing.T) {
//		contract.Require(t, []string{"../../crds"},
//			contract.Resource{Kind: "HPEStorageConnection", Role: contract.Connection},
//			contract.Resource{Kind: "HPEStorageClass", Role: contract.Class},
//		)
//	}
package contract

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// Role is what a kind is in the contract.
type Role string

const (
	Connection Role = "connection"
	Class      Role = "class"
)

// Categories of R6.
const (
	CategoryStorage     = "d8-storage"
	CategoryConnections = "d8-storage-connections"
	CategoryClasses     = "d8-storage-classes"
)

// Printer columns of R6.
const (
	ColumnReady      = "Ready"
	ColumnBackend    = "Backend"
	ColumnReason     = "Reason"
	ColumnAddress    = "Address"
	ColumnConnection = "Connection"
	ColumnPool       = "Pool"
	ColumnAge        = "Age"
	ColumnPhase      = "Phase"
)

// JSON paths the columns of R6 must read.
const (
	ReadyJSONPath   = `.status.conditions[?(@.type=="Ready")].status`
	ReasonJSONPath  = `.status.conditions[?(@.type=="Ready")].reason`
	BackendJSONPath = `.status.conditions[?(@.type=="BackendReady")].status`
	AgeJSONPath     = ".metadata.creationTimestamp"
)

// SensitiveDataKey marks a field whose value the API hides from a user without
// access to the sensitive subresource.
const SensitiveDataKey = "x-kubernetes-sensitive-data"

// SkippableColumns are the columns of R6 a kind may lack, through
// [Resource.SkipColumns].
var SkippableColumns = []string{ColumnBackend, ColumnAddress, ColumnConnection, ColumnPool}

var (
	credentialName = regexp.MustCompile(`^(password|secret|token)$|[a-z0-9](Password|Secret|Token|Key)$`)
	secretRefName  = regexp.MustCompile(`(?i)(secretname|secretref)$`)
)

// Resource is a kind of the module and what the contract expects of it.
type Resource struct {
	Kind string
	Role Role

	// SkipColumns are the columns of SkippableColumns the kind does not have,
	// such as Pool on a class of a storage without pools.
	SkipColumns []string

	// Sensitive are spec fields that hold credentials under a name the check
	// does not recognise, as dotted paths ("spec.auth.chap"; "[]" after an
	// array).
	Sensitive []string

	// NotSensitive are spec fields the check takes for credentials by their
	// name and that are not.
	NotSensitive []string
}

// Violation is one place where a CRD departs from the contract.
type Violation struct {
	Kind    string
	Version string
	Rule    string
	Message string
}

func (v Violation) String() string {
	if v.Version == "" {
		return fmt.Sprintf("%s: %s: %s", v.Kind, v.Rule, v.Message)
	}
	return fmt.Sprintf("%s %s: %s: %s", v.Kind, v.Version, v.Rule, v.Message)
}

// Require fails t with every violation of the CRDs found in crdPaths.
func Require(t testing.TB, crdPaths []string, resources ...Resource) {
	t.Helper()

	violations, err := Check(crdPaths, resources...)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations {
		t.Error(v)
	}
}

// Check reads the CRDs in crdPaths (files, or directories whose *.yaml and
// *.yml files are read, doc-ru-* skipped) and returns how the served versions
// of the kinds in resources depart from the contract. An error means the
// check could not run: a path or a kind not found, a manifest that does not
// parse, a Resource that is not valid.
func Check(crdPaths []string, resources ...Resource) ([]Violation, error) {
	if len(resources) == 0 {
		return nil, errors.New("contract: no resources")
	}

	crds, err := load(crdPaths)
	if err != nil {
		return nil, err
	}

	var violations []Violation
	for _, r := range resources {
		if err := r.validate(); err != nil {
			return nil, err
		}
		c, ok := crds[r.Kind]
		if !ok {
			return nil, fmt.Errorf("contract: no CRD of kind %s in %v", r.Kind, crdPaths)
		}
		violations = append(violations, r.check(c)...)
	}
	return violations, nil
}

func (r Resource) validate() error {
	if r.Kind == "" {
		return errors.New("contract: a resource without a kind")
	}
	if r.Role != Connection && r.Role != Class {
		return fmt.Errorf("contract: %s: role %q is neither %q nor %q", r.Kind, r.Role, Connection, Class)
	}
	for _, col := range r.SkipColumns {
		if !slices.Contains(SkippableColumns, col) {
			return fmt.Errorf("contract: %s: column %s cannot be skipped, only %v", r.Kind, col, SkippableColumns)
		}
	}
	return nil
}

func (r Resource) check(c *crd) []Violation {
	var out []Violation
	add := func(version, rule, format string, args ...any) {
		out = append(out, Violation{Kind: r.Kind, Version: version, Rule: rule, Message: fmt.Sprintf(format, args...)})
	}

	for _, cat := range r.categories() {
		if !slices.Contains(c.Spec.Names.Categories, cat) {
			add("", "R6", "category %s is missing from spec.names.categories %v", cat, c.Spec.Names.Categories)
		}
	}

	for _, short := range c.Spec.Names.ShortNames {
		owner, known := StorageShortNames[short]
		switch {
		case !known:
			add("", "R6", "short name %s is not in StorageShortNames; register it in sds-common-lib, where its uniqueness is checked", short)
		case owner != r.Kind:
			add("", "R6", "short name %s is already taken by %s", short, owner)
		}
	}

	for _, v := range c.Spec.Versions {
		if !v.Served {
			continue
		}
		r.checkColumns(v, func(format string, args ...any) { add(v.Name, "R6", format, args...) })
		checkStatus(v, func(format string, args ...any) { add(v.Name, "R4", format, args...) })
		r.checkCredentials(v, func(format string, args ...any) { add(v.Name, "R3", format, args...) })
	}
	return out
}

func (r Resource) categories() []string {
	if r.Role == Connection {
		return []string{CategoryStorage, CategoryConnections}
	}
	return []string{CategoryStorage, CategoryClasses}
}

func (r Resource) columns() []string {
	var cols []string
	if r.Role == Connection {
		cols = []string{ColumnReady, ColumnBackend, ColumnReason, ColumnAddress, ColumnAge}
	} else {
		cols = []string{ColumnReady, ColumnReason, ColumnConnection, ColumnPool, ColumnAge}
	}
	return slices.DeleteFunc(cols, func(col string) bool { return slices.Contains(r.SkipColumns, col) })
}

var columnJSONPaths = map[string]string{
	ColumnReady:   ReadyJSONPath,
	ColumnReason:  ReasonJSONPath,
	ColumnBackend: BackendJSONPath,
	ColumnAge:     AgeJSONPath,
}

func (r Resource) checkColumns(v version, add func(string, ...any)) {
	byName := make(map[string]apiextensionsv1.CustomResourceColumnDefinition, len(v.AdditionalPrinterColumns))
	for _, col := range v.AdditionalPrinterColumns {
		byName[strings.ToLower(col.Name)] = col
	}

	for _, name := range r.columns() {
		col, ok := byName[strings.ToLower(name)]
		if !ok {
			add("column %s is missing", name)
			continue
		}
		if col.Priority != 0 {
			add("column %s has priority %d, so kubectl get shows it only with -o wide", name, col.Priority)
		}
		if want, ok := columnJSONPaths[name]; ok && strings.ReplaceAll(col.JSONPath, "'", `"`) != want {
			add("column %s reads %s, not %s", name, col.JSONPath, want)
		}
		if name == ColumnAge && col.Type != "date" {
			add("column %s has type %s, not date", name, col.Type)
		}
	}

	if col, ok := byName[strings.ToLower(ColumnPhase)]; ok && col.Priority == 0 {
		add("column %s has priority 0; it is shown only with -o wide (priority: 1)", ColumnPhase)
	}
}

func checkStatus(v version, add func(string, ...any)) {
	if v.Subresources == nil || v.Subresources.Status == nil {
		add("the status subresource is not enabled")
	}

	conds := child(v.Schema.OpenAPIV3Schema, "status", "conditions")
	if conds == nil {
		add("status.conditions is missing, so the Ready condition has nowhere to be")
		return
	}
	if conds["type"] != "array" {
		add("status.conditions is of type %v, not array", conds["type"])
		return
	}
	item, _ := conds["items"].(map[string]any)
	for _, field := range []string{"type", "status", "reason", "message"} {
		if child(item, field) == nil {
			add("status.conditions[].%s is missing", field)
		}
	}
}

func (r Resource) checkCredentials(v version, add func(string, ...any)) {
	spec := child(v.Schema.OpenAPIV3Schema, "spec")
	if spec == nil {
		return
	}

	found := map[string]bool{}
	walk(spec, "spec", func(path, name string, node map[string]any) {
		found[path] = true
		if secretRefName.MatchString(name) {
			add("%s refers to a Secret; credentials are fields of the resource marked %s", path, SensitiveDataKey)
			return
		}
		credential := (credentialName.MatchString(name) || slices.Contains(r.Sensitive, path)) && !slices.Contains(r.NotSensitive, path)
		if credential && node[SensitiveDataKey] != true {
			add("%s holds credentials and is not marked %s: true", path, SensitiveDataKey)
		}
	})

	for _, path := range r.Sensitive {
		if !found[path] {
			add("%s is listed in Sensitive and is not in the schema", path)
		}
	}
}

func child(node map[string]any, names ...string) map[string]any {
	for _, name := range names {
		props, _ := node["properties"].(map[string]any)
		next, ok := props[name].(map[string]any)
		if !ok {
			return nil
		}
		node = next
	}
	return node
}

func walk(node map[string]any, path string, visit func(path, name string, node map[string]any)) {
	props, _ := node["properties"].(map[string]any)
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		next, ok := props[name].(map[string]any)
		if !ok {
			continue
		}
		p := path + "." + name
		visit(p, name, next)
		walk(next, p, visit)
	}

	if items, ok := node["items"].(map[string]any); ok {
		walk(items, path+"[]", visit)
	}
	if extra, ok := node["additionalProperties"].(map[string]any); ok {
		walk(extra, path+"{}", visit)
	}
}

type crd struct {
	Kind string `json:"kind"`
	Spec struct {
		Names    apiextensionsv1.CustomResourceDefinitionNames `json:"names"`
		Versions []version                                     `json:"versions"`
	} `json:"spec"`
}

type version struct {
	Name                     string                                           `json:"name"`
	Served                   bool                                             `json:"served"`
	Subresources             *apiextensionsv1.CustomResourceSubresources      `json:"subresources"`
	AdditionalPrinterColumns []apiextensionsv1.CustomResourceColumnDefinition `json:"additionalPrinterColumns"`
	Schema                   struct {
		OpenAPIV3Schema map[string]any `json:"openAPIV3Schema"`
	} `json:"schema"`
}

func load(paths []string) (map[string]*crd, error) {
	if len(paths) == 0 {
		return nil, errors.New("contract: no CRD paths")
	}

	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("contract: %w", err)
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil, fmt.Errorf("contract: %w", err)
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || strings.HasPrefix(name, "doc-ru-") || (filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml") {
				continue
			}
			files = append(files, filepath.Join(p, name))
		}
	}

	crds := map[string]*crd{}
	for _, f := range files {
		if err := loadFile(f, crds); err != nil {
			return nil, err
		}
	}
	return crds, nil
}

func loadFile(path string, crds map[string]*crd) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("contract: %w", err)
	}

	dec := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	for {
		c := &crd{}
		err := dec.Decode(c)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("contract: %s: %w", path, err)
		}
		if c.Kind != "CustomResourceDefinition" || c.Spec.Names.Kind == "" {
			continue
		}
		crds[c.Spec.Names.Kind] = c
	}
}
