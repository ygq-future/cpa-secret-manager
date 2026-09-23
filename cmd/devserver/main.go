// Command devserver runs a local CLIProxyAPI host simulator for the
// cpa-secret-manager plugin. It serves the production management page, the
// simulated host management API and the usage injection endpoints described in
// README.md.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type options struct {
	addr          string
	managementKey string
	keysPath      string
	statePath     string
}

func main() {
	parsed := parseOptions()

	host, err := newDevHost(devHostOptions{
		ManagementKey: parsed.managementKey,
		KeysPath:      parsed.keysPath,
		StatePath:     parsed.statePath,
	})
	if err != nil {
		log.Fatalf("devserver: %v", err)
	}

	server := &http.Server{
		Addr:              parsed.addr,
		Handler:           host.handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("devserver: control panel %s/control-panel", browserURL(parsed.addr))
		log.Printf("devserver: management key %q, key store %s, state %s", parsed.managementKey, parsed.keysPath, parsed.statePath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		if err != nil {
			log.Fatalf("devserver: %v", err)
		}
	case signalValue := <-signals:
		log.Printf("devserver: shutting down on %s", signalValue)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("devserver: http shutdown: %v", err)
	}
	if err := host.shutdown(shutdownCtx); err != nil {
		log.Printf("devserver: plugin shutdown: %v", err)
	}
}

func parseOptions() options {
	parsed := options{}
	flag.StringVar(&parsed.addr, "addr", envString("CPA_SECRET_MANAGER_DEVSERVER_ADDR", "127.0.0.1:8080"), "HTTP listen address; the default binds loopback only and does not trigger a firewall prompt")
	flag.StringVar(&parsed.managementKey, "management-key", envString("CPA_SECRET_MANAGER_DEVSERVER_KEY", "devkey"), "simulated management key")
	flag.StringVar(&parsed.keysPath, "keys", envString("CPA_SECRET_MANAGER_DEVSERVER_KEYS", "data/devserver/api-keys.json"), "simulated proxy API key store")
	flag.StringVar(&parsed.statePath, "state", envString("CPA_SECRET_MANAGER_DEVSERVER_STATE", "data/devserver/cache.json"), "plugin state document path")
	flag.Parse()
	return parsed
}

func envString(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// browserURL renders the listen address as a URL a user can open. Wildcard
// hosts are shown as localhost so the printed link always works locally.
func browserURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}
