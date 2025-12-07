// ABOUTME: Entry point for the ISH fake Google API server.
// ABOUTME: Wires together store, auth, and API handlers with CLI commands.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/cobra"
	"github.com/2389/ish/internal/admin"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/logging"
	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	_ "github.com/2389/ish/plugins/discord"  // Register Discord plugin
	_ "github.com/2389/ish/plugins/github"   // Register GitHub plugin
	_ "github.com/2389/ish/plugins/google"   // Register Google plugin
	_ "github.com/2389/ish/plugins/oauth"    // Register OAuth plugin
	_ "github.com/2389/ish/plugins/sendgrid" // Register SendGrid plugin
	_ "github.com/2389/ish/plugins/twilio"   // Register Twilio plugin
)

var (
	port   string
	dbPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ish",
		Short: "Fake Google API server",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE:  runServe,
	}
	serveCmd.Flags().StringVarP(&port, "port", "p", getEnv("ISH_PORT", "9000"), "Port to listen on")
	serveCmd.Flags().StringVarP(&dbPath, "db", "d", getEnv("ISH_DB_PATH", "./ish.db"), "Database path")

	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with test data",
		RunE:  runSeed,
	}
	seedCmd.Flags().StringVarP(&dbPath, "db", "d", getEnv("ISH_DB_PATH", "./ish.db"), "Database path")

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset the database (wipe and reseed)",
		RunE:  runReset,
	}
	resetCmd.Flags().StringVarP(&dbPath, "db", "d", getEnv("ISH_DB_PATH", "./ish.db"), "Database path")

	rootCmd.AddCommand(serveCmd, seedCmd, resetCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	srv, err := newServer(dbPath)
	if err != nil {
		return err
	}

	addr := ":" + port
	log.Printf("ISH server listening on %s", addr)
	log.Printf("Database: %s", dbPath)
	return http.ListenAndServe(addr, srv)
}

func newServer(dbPath string) (http.Handler, error) {
	s, err := store.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open store: %w", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(logging.Middleware(s))
	r.Use(auth.Middleware)

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	// Favicon
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Initialize all plugins with database access
	for _, plugin := range core.All() {
		// Set database for plugins that need it
		if dbPlugin, ok := plugin.(core.DatabasePlugin); ok {
			if err := dbPlugin.SetDB(s.GetDB()); err != nil {
				return nil, fmt.Errorf("failed to initialize plugin %s: %w", plugin.Name(), err)
			}
		}

		// Register routes
		plugin.RegisterAuth(r)
		plugin.RegisterRoutes(r)
	}

	// Admin UI
	admin.NewHandlers(s).RegisterRoutes(r)

	return r, nil
}

func runSeed(cmd *cobra.Command, args []string) error {
	s, err := store.New(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	return seedData(s)
}

func runReset(cmd *cobra.Command, args []string) error {
	// Remove existing database
	os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	return seedData(s)
}

func seedData(s *store.Store) error {
	// TODO: Update to work with plugin architecture
	// Each plugin should handle its own seeding via the Plugin.Seed() method
	log.Println("Seed functionality not yet implemented with plugin architecture")
	log.Println("Use plugin-specific seeding methods instead")
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
