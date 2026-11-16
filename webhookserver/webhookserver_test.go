package webhookserver

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type fakeManager struct {
	runnables []manager.Runnable
	checks    map[string]healthz.Checker
	addErr    error
}

func (m *fakeManager) Add(r manager.Runnable) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.runnables = append(m.runnables, r)
	return nil
}

func (m *fakeManager) AddReadyzCheck(name string, check healthz.Checker) error {
	if m.checks == nil {
		m.checks = map[string]healthz.Checker{}
	}
	m.checks[name] = check
	return nil
}

func TestOptionsFromEnv(t *testing.T) {
	cases := []struct {
		name    string
		addr    string
		certDir string
		enabled bool
		wantErr bool
		want    webhook.Options
	}{
		{name: "unset", enabled: false},
		{name: "port only", addr: ":8443", certDir: "/certs", enabled: true, want: webhook.Options{Port: 8443, CertDir: "/certs"}},
		{name: "host and port", addr: "127.0.0.1:9443", certDir: "/certs", enabled: true, want: webhook.Options{Host: "127.0.0.1", Port: 9443, CertDir: "/certs"}},
		{name: "no port", addr: "8443", certDir: "/certs", wantErr: true},
		{name: "bad port", addr: ":http", certDir: "/certs", wantErr: true},
		{name: "port out of range", addr: ":70000", certDir: "/certs", wantErr: true},
		{name: "no cert dir", addr: ":8443", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(BindAddressEnvName, tc.addr)
			t.Setenv(CertDirEnvName, tc.certDir)
			got, enabled, err := OptionsFromEnv()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got options %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if enabled != tc.enabled {
				t.Fatalf("enabled = %v, want %v", enabled, tc.enabled)
			}
			if got.Host != tc.want.Host || got.Port != tc.want.Port || got.CertDir != tc.want.CertDir {
				t.Fatalf("options = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestAddFromEnvDisabled(t *testing.T) {
	t.Setenv(BindAddressEnvName, "")
	mgr := &fakeManager{}
	enabled, err := AddFromEnv(mgr, map[string]http.Handler{"/x": http.NotFoundHandler()})
	if err != nil || enabled {
		t.Fatalf("AddFromEnv = %v, %v; want false, nil", enabled, err)
	}
	if len(mgr.runnables) != 0 || len(mgr.checks) != 0 {
		t.Fatalf("nothing must be added when disabled")
	}
}

func TestAddRejectsBadInput(t *testing.T) {
	if err := Add(&fakeManager{}, webhook.Options{Port: 1}, nil); err == nil {
		t.Fatal("expected an error for no handlers")
	}
	if err := Add(&fakeManager{}, webhook.Options{Port: 1}, map[string]http.Handler{"/x": nil}); err == nil {
		t.Fatal("expected an error for a nil handler")
	}
	if err := Add(&fakeManager{addErr: errors.New("boom")}, webhook.Options{Port: 1}, map[string]http.Handler{"/x": http.NotFoundHandler()}); err == nil {
		t.Fatal("expected the manager error to be returned")
	}
}

func TestAddServesHandlersAndReportsReadiness(t *testing.T) {
	certDir := t.TempDir()
	writeSelfSignedCert(t, certDir)
	port := freePort(t)
	t.Setenv(BindAddressEnvName, fmt.Sprintf("127.0.0.1:%d", port))
	t.Setenv(CertDirEnvName, certDir)

	mgr := &fakeManager{}
	enabled, err := AddFromEnv(mgr, map[string]http.Handler{
		"/sc-validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "validated")
		}),
	})
	if err != nil || !enabled {
		t.Fatalf("AddFromEnv = %v, %v; want true, nil", enabled, err)
	}
	if len(mgr.runnables) != 1 {
		t.Fatalf("runnables = %d, want 1", len(mgr.runnables))
	}
	if r, ok := mgr.runnables[0].(manager.LeaderElectionRunnable); !ok || r.NeedLeaderElection() {
		t.Fatal("the webhook server must run on every replica, not only on the leader")
	}
	check := mgr.checks[ReadyzCheckName]
	if check == nil {
		t.Fatalf("readiness check %q not registered", ReadyzCheckName)
	}
	if err := check(nil); err == nil {
		t.Fatal("readiness must fail before the server starts")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- mgr.runnables[0].Start(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("webhook server stopped with an error: %v", err)
		}
	})

	deadline := time.Now().Add(10 * time.Second)
	for check(nil) != nil {
		if time.Now().After(deadline) {
			t.Fatalf("server not ready: %v", check(nil))
		}
		time.Sleep(50 * time.Millisecond)
	}

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec
	resp, err := client.Post(fmt.Sprintf("https://127.0.0.1:%d/sc-validate", port), "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "validated" {
		t.Fatalf("response = %d %q", resp.StatusCode, body)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func writeSelfSignedCert(t *testing.T, dir string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "webhooks"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tls.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tls.key"), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
}
