package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"ssc/core"
	"strings"
	"sync"
	"testing"
	"time"
)

type readyWriter struct{ ready chan string }

func (w readyWriter) Write(p []byte) (int, error) {
	w.ready <- strings.TrimSpace(strings.TrimPrefix(string(p), "Serving SSC on "))
	return len(p), nil
}

func startServer(t *testing.T, dataDir, configPath string, extra ...string) (string, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, append([]string{"--config", configPath, "--data", dataDir, "--listen", "127.0.0.1:0"}, extra...), readyWriter{ready})
	}()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("server shutdown: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Error("server did not shut down")
			}
		})
	}
	select {
	case url := <-ready:
		t.Cleanup(stop)
		return url, stop
	case err := <-done:
		cancel()
		t.Fatalf("server startup: %v", err)
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("server startup timed out")
	}
	return "", func() {}
}

func TestRunningServerPersistenceAccessAndCompetingWriters(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "server.json")
	dataDir := filepath.Join(root, "data")
	config := testConfig()
	data, _ := json.Marshal(config)
	if err := ioutil.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	url, stop := startServer(t, dataDir, configPath)
	// A second server cannot bypass the first server's in-process ref locks.
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"--config", configPath, "--data", dataDir, "--listen", "127.0.0.1:0"}, &out); err == nil {
		t.Fatal("second storage owner accepted")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	call := func(method, route, token, contentType, kind string, body []byte) (int, []byte, error) {
		req, err := http.NewRequest(method, url+route, bytes.NewReader(body))
		if err != nil {
			return 0, nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if kind != "" {
			req.Header.Set("X-SSC-Object-Type", kind)
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer resp.Body.Close()
		response, err := ioutil.ReadAll(resp.Body)
		return resp.StatusCode, response, err
	}
	upload := func(kind string, body []byte) string {
		t.Helper()
		hash, _, err := core.InspectObject(kind, body)
		if err != nil {
			t.Fatal(err)
		}
		status, data, err := call("PUT", "/v1/repos/demo/objects/"+hash, "write-secret", "application/octet-stream", kind, body)
		if err != nil || status != 204 {
			t.Fatalf("upload %d %s %v", status, data, err)
		}
		return hash
	}
	blob := upload("blob", []byte("hello from a teammate"))
	tree := upload("tree", []byte("hello.txt "+blob))
	first := upload("commit", commitBody(tree, "first"))
	status, body, err := call("PUT", "/v1/repos/demo/refs/main", "write-secret", "application/json", "", []byte(fmt.Sprintf(`{"old":"","new":%q}`, first)))
	if err != nil || status != 200 {
		t.Fatalf("create ref %d %s %v", status, body, err)
	}
	next := []string{upload("commit", commitBody(tree, "alice", first)), upload("commit", commitBody(tree, "bob", first))}
	results := make(chan int, 2)
	failures := make(chan error, 2)
	var wg sync.WaitGroup
	for _, tip := range next {
		wg.Add(1)
		go func(tip string) {
			defer wg.Done()
			code, _, err := call("PUT", "/v1/repos/demo/refs/main", "write-secret", "application/json", "", []byte(fmt.Sprintf(`{"old":%q,"new":%q}`, first, tip)))
			if err != nil {
				failures <- err
			}
			results <- code
		}(tip)
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	codes := map[int]int{}
	for code := range results {
		codes[code]++
	}
	if codes[200] != 1 || codes[409] != 1 {
		t.Fatal(codes)
	}
	status, body, err = call("GET", "/v1/repos/demo/refs/main", "read-secret", "", "", nil)
	if err != nil || status != 200 {
		t.Fatal(status, string(body), err)
	}
	persisted := string(body)
	stop()
	if _, err := os.Stat(filepath.Join(dataDir, ".server.lock")); !os.IsNotExist(err) {
		t.Fatal("lock not released")
	}
	// Restart revokes write permission and reloads the same objects and refs.
	config.Tokens[0].Repositories["demo"] = "read"
	data, _ = json.Marshal(config)
	if err := ioutil.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	url, stop = startServer(t, dataDir, configPath)
	defer stop()
	status, body, err = call("GET", "/v1/repos/demo/refs/main", "read-secret", "", "", nil)
	if err != nil || status != 200 || string(body) != persisted {
		t.Fatal("history did not survive restart", status, string(body), err)
	}
	status, body, err = call("GET", "/v1/repos/demo/objects/"+blob, "read-secret", "", "", nil)
	if err != nil || status != 200 || string(body) != "hello from a teammate" {
		t.Fatal(status, string(body), err)
	}
	status, _, err = call("GET", "/v1/repos/private/objects/"+blob, "read-secret", "", "", nil)
	if err != nil || status != 404 {
		t.Fatal("repository isolation failed", status, err)
	}
	status, _, err = call("PUT", "/v1/repos/demo/objects/"+blob, "write-secret", "application/octet-stream", "blob", []byte("hello from a teammate"))
	if err != nil || status != 403 {
		t.Fatal("permission revocation failed", status, err)
	}
}

type brokenReader struct{ sent bool }

func (r *brokenReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, []byte("partial")), nil
	}
	return 0, io.ErrUnexpectedEOF
}

func TestTLSListener(t *testing.T) {
	root := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "SSC test"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	certPath, keyPath, configPath := filepath.Join(root, "cert.pem"), filepath.Join(root, "key.pem"), filepath.Join(root, "server.json")
	config, _ := json.Marshal(testConfig())
	for path, data := range map[string][]byte{certPath: certPEM, keyPath: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKey}), configPath: config} {
		if err := ioutil.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	url, _ := startServer(t, filepath.Join(root, "data"), configPath, "--tls-cert", certPath, "--tls-key", keyPath)
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certPEM) {
		t.Fatal("invalid test certificate")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	response, err := client.Get(url + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || response.TLS == nil {
		t.Fatal("TLS listener failed")
	}
}
