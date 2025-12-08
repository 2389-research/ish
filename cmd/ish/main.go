// ABOUTME: Entry point for the ISH fake Google API server.
// ABOUTME: Wires together store, auth, and API handlers with CLI commands.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/cobra"
	"github.com/2389/ish/internal/admin"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/logging"
	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	_ "github.com/2389/ish/plugins/discord"       // Register Discord plugin
	_ "github.com/2389/ish/plugins/github"        // Register GitHub plugin
	_ "github.com/2389/ish/plugins/google"        // Register Google plugin
	_ "github.com/2389/ish/plugins/homeassistant" // Register Home Assistant plugin
	_ "github.com/2389/ish/plugins/oauth"         // Register OAuth plugin
	_ "github.com/2389/ish/plugins/sendgrid"      // Register SendGrid plugin
	_ "github.com/2389/ish/plugins/twilio"        // Register Twilio plugin
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
	serveCmd.Flags().StringVarP(&dbPath, "db", "d", getDefaultDBPath(), "Database path")

	seedCmd := &cobra.Command{
		Use:   "seed [plugin]",
		Short: "Seed the database with test data (optionally for a specific plugin)",
		Long:  "Seed the database with test data. If no plugin is specified, seeds all plugins. Use 'seed <plugin-name>' to seed only a specific plugin.",
		RunE:  runSeed,
		Args:  cobra.MaximumNArgs(1),
	}
	seedCmd.Flags().StringVarP(&dbPath, "db", "d", getDefaultDBPath(), "Database path")

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset the database (wipe and reseed)",
		RunE:  runReset,
	}
	resetCmd.Flags().StringVarP(&dbPath, "db", "d", getDefaultDBPath(), "Database path")

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

	var pluginName string
	if len(args) > 0 {
		pluginName = args[0]
	}

	return seedData(s, pluginName)
}

func runReset(cmd *cobra.Command, args []string) error {
	// Remove existing database
	os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	return seedData(s, "") // Reset always seeds all plugins
}

func seedData(s *store.Store, pluginFilter string) error {
	if pluginFilter != "" {
		log.Printf("Seeding database with test data for plugin: %s", pluginFilter)
	} else {
		log.Println("Seeding database with test data...")
	}

	// Initialize all plugins with database access
	for _, plugin := range core.All() {
		if dbPlugin, ok := plugin.(core.DatabasePlugin); ok {
			if err := dbPlugin.SetDB(s.GetDB()); err != nil {
				log.Printf("Failed to initialize plugin %s: %v", plugin.Name(), err)
				continue
			}
		}
	}

	// Seed each plugin (optionally filtered by name)
	totalRecords := 0
	seededCount := 0
	for _, plugin := range core.All() {
		// Skip if filter is set and doesn't match
		if pluginFilter != "" && plugin.Name() != pluginFilter {
			continue
		}

		seedData, err := plugin.Seed(context.Background(), "medium")
		if err != nil {
			log.Printf("❌ Failed to seed %s: %v", plugin.Name(), err)
			continue
		}

		if seedData.Summary != "" && seedData.Summary != "Not yet implemented" {
			log.Printf("✅ %s: %s", plugin.Name(), seedData.Summary)
			for _, count := range seedData.Records {
				totalRecords += count
			}
			seededCount++
		}
	}

	// Check if plugin filter didn't match anything
	if pluginFilter != "" && seededCount == 0 {
		log.Printf("❌ Plugin '%s' not found or has no seed implementation", pluginFilter)
		log.Println("\nAvailable plugins:")
		for _, plugin := range core.All() {
			log.Printf("  - %s", plugin.Name())
		}
		return fmt.Errorf("plugin '%s' not found", pluginFilter)
	}

	if pluginFilter != "" {
		log.Printf("\n🎉 Seeding complete! Created %d records for %s", totalRecords, pluginFilter)
	} else {
		log.Printf("\n🎉 Seeding complete! Created %d total records across all plugins", totalRecords)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// getDefaultDBPath returns the default database path following XDG Base Directory spec
// Priority: ISH_DB_PATH env var > ./ish.db (backwards compat) > XDG_DATA_HOME/ish/ish.db
func getDefaultDBPath() string {
	// 1. Check environment variable first
	if envPath := os.Getenv("ISH_DB_PATH"); envPath != "" {
		return envPath
	}

	// 2. Check for existing ./ish.db (backwards compatibility)
	cwdPath := "./ish.db"
	if _, err := os.Stat(cwdPath); err == nil {
		return cwdPath
	}

	// 3. Use XDG Base Directory spec
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		// Default to ~/.local/share per XDG spec
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if we can't get home dir
			return cwdPath
		}
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	ishDataDir := filepath.Join(dataHome, "ish")
	dbPath := filepath.Join(ishDataDir, "ish.db")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(ishDataDir, 0755); err != nil {
		log.Printf("Warning: Could not create XDG data directory %s: %v", ishDataDir, err)
		return cwdPath
	}

	return dbPath
}
