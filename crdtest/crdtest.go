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

// Package crdtest runs a module's tests of its CRDs against a real
// kube-apiserver (envtest). The apiserver installs the generated manifests and
// then admits or rejects objects the way a cluster does: the OpenAPI schema,
// the defaults, the CEL rules with their cost budget and the transition rules
// of an update are all applied, which a validator built from a single schema
// node does not reproduce.
//
// A test package starts one environment for all of its tests:
//
//	var k8sClient client.Client
//
//	func TestMain(m *testing.M) {
//		os.Exit(crdtest.Main(m, crdtest.Options{
//			CRDPaths:    []string{"../../crds"},
//			AddToScheme: []func(*runtime.Scheme) error{v1alpha1.AddToScheme},
//		}, func(env *crdtest.Env) { k8sClient = env.Client }))
//	}
package crdtest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// DefaultKubernetesVersion is the kube-apiserver the CRDs are installed
	// into unless Options.KubernetesVersion or KubernetesVersionEnv says
	// otherwise. It is a setup-envtest selector, so the latest patch release is
	// taken.
	DefaultKubernetesVersion = "1.33.x"

	// KubernetesVersionEnv overrides the version of every test package without
	// a code change, for running the same suite against another apiserver.
	KubernetesVersionEnv = "ENVTEST_K8S_VERSION"

	// AssetsEnv points at kube-apiserver and etcd binaries already on disk.
	// When it is set nothing is downloaded and the version is whatever those
	// binaries are.
	AssetsEnv = "KUBEBUILDER_ASSETS"

	// setupEnvtest is run with `go run` rather than imported, so its
	// dependencies never enter the build list of the module under test.
	setupEnvtest = "sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.25.2"
)

// Options describes the CRDs to install and the types the client must know.
type Options struct {
	// CRDPaths are CRD manifests or directories holding them, relative to the
	// test package. Files in a directory that are not CRDs, such as the
	// doc-ru-*.yaml documentation of a Deckhouse module, are skipped. A missing
	// path is an error.
	CRDPaths []string

	// AddToScheme registers the module's API types with Env.Client, next to the
	// client-go types.
	AddToScheme []func(*runtime.Scheme) error

	// KubernetesVersion is a setup-envtest selector, DefaultKubernetesVersion
	// when empty. KubernetesVersionEnv takes precedence over it.
	KubernetesVersion string
}

// Env is a running kube-apiserver with the CRDs installed.
type Env struct {
	Config *rest.Config
	Client client.Client

	testEnv *envtest.Environment
}

// Start starts kube-apiserver and etcd and installs the CRDs. It replaces the
// controller-runtime logger with a discarding one, because the logs of envtest
// itself say nothing about the CRDs under test.
func Start(opts Options) (*Env, error) {
	if len(opts.CRDPaths) == 0 {
		return nil, errors.New("crdtest: no CRDPaths")
	}

	logf.SetLogger(logr.Discard())

	assets, err := assetsDir(opts.KubernetesVersion)
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("crdtest: register client-go types: %w", err)
	}
	for _, add := range opts.AddToScheme {
		if err := add(scheme); err != nil {
			return nil, fmt.Errorf("crdtest: register API types: %w", err)
		}
	}

	testEnv := &envtest.Environment{
		BinaryAssetsDirectory: assets,
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths:  opts.CRDPaths,
			Scheme: scheme,
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return nil, fmt.Errorf("crdtest: start kube-apiserver: %w", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, errors.Join(fmt.Errorf("crdtest: create client: %w", err), testEnv.Stop())
	}

	return &Env{Config: cfg, Client: cl, testEnv: testEnv}, nil
}

// Stop stops kube-apiserver and etcd.
func (e *Env) Stop() error {
	if err := e.testEnv.Stop(); err != nil {
		return fmt.Errorf("crdtest: stop kube-apiserver: %w", err)
	}
	return nil
}

// Main is the body of a TestMain: it starts the environment, hands it to
// ready, runs the tests and stops it. It returns the exit code for os.Exit,
// which is 1 when the environment fails to start or to stop.
func Main(m *testing.M, opts Options, ready func(*Env)) int {
	env, err := Start(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	ready(env)
	code := m.Run()

	if err := env.Stop(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

func assetsDir(version string) (string, error) {
	if dir := os.Getenv(AssetsEnv); dir != "" {
		return dir, nil
	}

	if v := os.Getenv(KubernetesVersionEnv); v != "" {
		version = v
	}
	if version == "" {
		version = DefaultKubernetesVersion
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", setupEnvtest, "use", version, "-p", "path")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("crdtest: setup-envtest use %s: %w\n%s", version, err, stderr.String())
	}

	dir := strings.TrimSpace(stdout.String())
	if dir == "" {
		return "", fmt.Errorf("crdtest: setup-envtest use %s printed no path\n%s", version, stderr.String())
	}
	return dir, nil
}
