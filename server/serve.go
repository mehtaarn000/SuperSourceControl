package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Run serves until ctx is canceled. No local .ssc repository/config is required.
func Run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("ssc serve", flag.ContinueOnError)
	flags.SetOutput(out)
	configPath := flags.String("config", "", "server JSON configuration path (required)")
	dataDir := flags.String("data", "", "server storage directory (required)")
	address := flags.String("listen", "127.0.0.1:8080", "listen address; use TLS for remote access")
	certFile := flags.String("tls-cert", "", "TLS certificate file")
	keyFile := flags.String("tls-key", "", "TLS private key file")
	generate := flags.Bool("generate-token", false, "print a random token and its SHA-256 digest, then exit")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *generate {
		if len(args) != 1 {
			return fmt.Errorf("use --generate-token by itself")
		}
		token := make([]byte, 32)
		if _, err := rand.Read(token); err != nil {
			return err
		}
		secret := hex.EncodeToString(token)
		digest := sha256.Sum256([]byte(secret))
		return json.NewEncoder(out).Encode(map[string]string{"token": secret, "sha256": hex.EncodeToString(digest[:])})
	}
	if *configPath == "" || *dataDir == "" {
		return fmt.Errorf("--config and --data are required")
	}
	if (*certFile == "") != (*keyFile == "") {
		return fmt.Errorf("--tls-cert and --tls-key must be used together")
	}
	host, _, err := net.SplitHostPort(*address)
	if err != nil {
		return err
	}
	if *certFile == "" && !isLoopback(host) {
		return fmt.Errorf("non-loopback listeners require --tls-cert and --tls-key; a TLS reverse proxy may target a loopback listener")
	}
	// Load the certificate before acquiring storage or opening the listener.
	var tlsConfig *tls.Config
	if *certFile != "" {
		certificate, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			return err
		}
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}
	}
	config, err := LoadConfig(*configPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*dataDir, 0700); err != nil {
		return err
	}
	lockPath := filepath.Join(*dataDir, ".server.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("cannot acquire server storage lock: %w", err)
	}
	defer os.Remove(lockPath)
	if _, err := fmt.Fprintf(lock, "%d\n", os.Getpid()); err != nil {
		lock.Close()
		return err
	}
	if err := lock.Close(); err != nil {
		return err
	}
	store, err := NewStore(*dataDir, config.Repositories)
	if err != nil {
		return err
	}
	handler, err := NewHandler(store, config)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return err
	}
	defer listener.Close()
	scheme := "http"
	if tlsConfig != nil {
		listener = tls.NewListener(listener, tlsConfig)
		scheme = "https"
	}
	srv := &http.Server{
		Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	finished := make(chan error, 1)
	go func() { finished <- srv.Serve(listener) }()
	fmt.Fprintf(out, "Serving SSC on %s://%s\n", scheme, listener.Addr())
	select {
	case err := <-finished:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			srv.Close()
			<-finished
			return err
		}
		err := <-finished
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func isLoopback(host string) bool {
	// An explicit IP avoids DNS changes exposing a plaintext token listener.
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
