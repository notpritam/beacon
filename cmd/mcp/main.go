// ABOUTME: beacon-mcp serves Beacon's MCP tools to Wingman over streamable HTTP.
// ABOUTME: Wires config -> store -> mcptools -> mcp.Server, behind a bearer-token check.

// Command beaconmcp runs the Beacon MCP server.
package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/notpritam/beacon/internal/config"
	"github.com/notpritam/beacon/internal/mcptools"
	"github.com/notpritam/beacon/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.WingmanToken == "" {
		return fmt.Errorf("BEACON_WINGMAN_TOKEN is required to run the MCP server")
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	tools := mcptools.New(st, mcptools.Options{})
	server := mcp.NewServer(&mcp.Implementation{Name: "beacon", Version: "v0.1.0"}, nil)
	registerTools(server, tools)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", bearerAuth(cfg.WingmanToken, handler))

	srv := &http.Server{
		Addr:              cfg.MCPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("beacon-mcp listening on %s/mcp\n", cfg.MCPAddr)
	return srv.ListenAndServe()
}

// bearerAuth requires Authorization: Bearer <token> (constant-time compare).
func bearerAuth(token string, next http.Handler) http.Handler {
	want := "Bearer " + token
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
