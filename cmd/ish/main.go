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
	"runtime"
	"strings"

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
		Short: "ISH - Intelligent Server Hub: Mock API server for testing",
		Long: `ISH (Intelligent Server Hub) is a comprehensive mock API server for development and testing.

Supports 7+ popular APIs including:
  • Google (Gmail, Calendar, People, Tasks)
  • GitHub (repos, issues, PRs, webhooks)
  • Twilio (SMS, calls)
  • Discord (webhooks)
  • SendGrid (email)
  • Home Assistant (smart home)
  • OAuth 2.0 (authorization flows)

Features:
  • AI-powered realistic test data generation
  • Google API Discovery Service support
  • SQLite persistence for stateful testing
  • Auto-generated admin UI
  • Full query/filtering support
  • 99+ API endpoints

Quick Start:
  ish seed          # Generate test data
  ish serve         # Start server on port 9000
  ish reset         # Wipe and reseed database`,
	}

	// Calculate default database path once (not per-command)
	defaultDBPath := getDefaultDBPath()

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		Long: `Start the ISH HTTP server on the specified port.

The server provides:
  • All API endpoints for supported plugins
  • Admin UI at http://localhost:PORT/admin
  • Health check at http://localhost:PORT/healthz
  • Google API Discovery Service

Authentication:
  Use Bearer tokens in the format: Bearer user:USERNAME
  Example: curl -H "Authorization: Bearer user:me" http://localhost:9000/gmail/v1/users/me/messages

Environment Variables:
  ISH_PORT          Server port (default: 9000)
  OPENAI_API_KEY    Enable AI-powered features
  ISH_AUTO_REPLY    Enable auto-reply (true/false)`,
		RunE: runServe,
	}
	serveCmd.Flags().StringVarP(&port, "port", "p", getEnv("ISH_PORT", "9000"), "Port to listen on")
	serveCmd.Flags().StringVarP(&dbPath, "db", "d", defaultDBPath, "Database path")

	seedCmd := &cobra.Command{
		Use:   "seed [plugin]",
		Short: "Seed the database with test data",
		Long: `Seed the database with realistic test data for all plugins or a specific one.

AI-Powered Generation:
  Set OPENAI_API_KEY to use AI for generating realistic emails, events, and contacts.
  Falls back to static test data if no API key is provided.

Usage:
  ish seed              # Seed all plugins with test data
  ish seed google       # Seed only Google plugin
  ish seed github       # Seed only GitHub plugin

Available Plugins:
  google, github, twilio, discord, sendgrid, homeassistant, oauth

Data Generated:
  • Gmail: 8 messages, threads, labels
  • Calendar: 5 events with realistic times
  • People: 5 contacts with names and emails
  • Tasks: 5 tasks in multiple lists
  • GitHub: Users, repos, issues, PRs, comments
  • Twilio: Phone numbers, SMS messages, calls
  • SendGrid: Email accounts, API keys, messages
  • Home Assistant: Devices, entities, states

Note: Seed is not idempotent. Use 'ish reset' to clear data before reseeding.`,
		RunE: runSeed,
		Args: cobra.MaximumNArgs(1),
	}
	seedCmd.Flags().StringVarP(&dbPath, "db", "d", defaultDBPath, "Database path")

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset the database (wipe and reseed)",
		Long: `Delete the database file and create a fresh one with new test data.

This command:
  1. Deletes the existing database file
  2. Creates a new empty database
  3. Seeds it with fresh test data for all plugins

Use this when:
  • You need to start fresh with clean data
  • Seed data has become inconsistent
  • You want to regenerate AI-powered data with different results
  • You accidentally corrupted the database

Warning: This permanently deletes all data in the database!`,
		RunE: runReset,
	}
	resetCmd.Flags().StringVarP(&dbPath, "db", "d", defaultDBPath, "Database path")

	rootCmd.AddCommand(serveCmd, seedCmd, resetCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// validateAndCleanDBPath validates and cleans a database path.
// Handles Unix/Linux, macOS, and Windows paths (including UNC and drive letters).
func validateAndCleanDBPath(path string) (string, error) {
	cleanPath := strings.TrimSpace(path)
	cleanPath = filepath.Clean(cleanPath)

	// Reject empty and root-like paths
	if cleanPath == "" || cleanPath == "." || cleanPath == "/" {
		return "", fmt.Errorf("database path cannot be empty, '.', or '/'")
	}

	// Windows: reject bare drive letters (e.g., "C:", "D:")
	if runtime.GOOS == "windows" && len(cleanPath) == 2 && cleanPath[1] == ':' {
		return "", fmt.Errorf("database path cannot be a bare drive letter")
	}

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("database path cannot contain '..'")
	}

	// Reject known problematic patterns
	badPatterns := []string{
		".git",
		".svn",
		"node_modules",
		".env",
		"credentials",
		"secret",
	}
	lowerPath := strings.ToLower(cleanPath)
	for _, pattern := range badPatterns {
		if strings.Contains(lowerPath, pattern) {
			return "", fmt.Errorf("database path cannot contain '%s' directory", pattern)
		}
	}

	return cleanPath, nil
}

func runServe(cmd *cobra.Command, args []string) error {
	var err error
	dbPath, err = validateAndCleanDBPath(dbPath)
	if err != nil {
		return err
	}

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
	var err error
	dbPath, err = validateAndCleanDBPath(dbPath)
	if err != nil {
		return err
	}

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
	var err error
	dbPath, err = validateAndCleanDBPath(dbPath)
	if err != nil {
		return err
	}

	// Remove existing database - ignore if file doesn't exist
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing database: %w", err)
	}

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
	hasUniqueError := false
	for _, plugin := range core.All() {
		// Skip if filter is set and doesn't match
		if pluginFilter != "" && plugin.Name() != pluginFilter {
			continue
		}

		seedData, err := plugin.Seed(context.Background(), "medium")
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "UNIQUE constraint failed") {
				log.Printf("Failed to seed %s: %v", plugin.Name(), err)
				hasUniqueError = true
			} else {
				log.Printf("Failed to seed %s: %v", plugin.Name(), err)
			}
			continue
		}

		if seedData.Summary != "" && seedData.Summary != "Not yet implemented" {
			log.Printf("%s: %s", plugin.Name(), seedData.Summary)
			for _, count := range seedData.Records {
				totalRecords += count
			}
			seededCount++
		}
	}

	// Show helpful message if UNIQUE constraint errors occurred
	if hasUniqueError {
		log.Println("\nNote: Database already contains seed data. Use 'ish reset' to clear and reseed.")
	}

	// Check if plugin filter didn't match anything
	if pluginFilter != "" && seededCount == 0 {
		log.Printf("Plugin '%s' not found or has no seed implementation", pluginFilter)
		log.Println("\nAvailable plugins:")
		for _, plugin := range core.All() {
			log.Printf("  - %s", plugin.Name())
		}
		return fmt.Errorf("plugin '%s' not found", pluginFilter)
	}

	if pluginFilter != "" {
		log.Printf("\nSeeding complete! Created %d records for %s", totalRecords, pluginFilter)
	} else {
		log.Printf("\nSeeding complete! Created %d total records across all plugins", totalRecords)
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
		// Trim whitespace and clean path
		envPath = strings.TrimSpace(envPath)
		envPath = filepath.Clean(envPath)
		if envPath == "" || envPath == "." {
			log.Printf("Warning: ISH_DB_PATH is invalid (empty or '.'), using default path")
		} else {
			return envPath
		}
	}

	// 2. Check for existing ./ish.db (backwards compatibility)
	cwdPath := "./ish.db"
	if _, err := os.Stat(cwdPath); err == nil {
		return cwdPath
	}

	// 3. Use XDG Base Directory spec (or Windows equivalent)
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil || homeDir == "" || homeDir == "/" {
			// Fallback to current directory if we can't get valid home dir
			log.Printf("Warning: Could not determine valid home directory (%q): %v, using ./ish.db", homeDir, err)
			return cwdPath
		}

		// Use platform-appropriate data directory
		// Windows: %LOCALAPPDATA% or ~/AppData/Local
		// Unix/Linux/macOS: ~/.local/share (XDG spec)
		if runtime.GOOS == "windows" {
			dataHome = os.Getenv("LOCALAPPDATA")
			if dataHome == "" {
				dataHome = filepath.Join(homeDir, "AppData", "Local")
			}
		} else {
			// Unix/Linux/macOS - XDG Base Directory spec
			dataHome = filepath.Join(homeDir, ".local", "share")
		}
	}

	ishDataDir := filepath.Join(dataHome, "ish")
	xdgDBPath := filepath.Join(ishDataDir, "ish.db")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(ishDataDir, 0755); err != nil {
		log.Printf("Warning: Could not create data directory %s: %v, using ./ish.db", ishDataDir, err)
		return cwdPath
	}

	// Verify we can write to the directory
	testFile := filepath.Join(ishDataDir, ".write-test")
	if f, err := os.Create(testFile); err != nil {
		log.Printf("Warning: Cannot write to data directory %s: %v, using ./ish.db", ishDataDir, err)
		return cwdPath
	} else {
		if err := f.Close(); err != nil {
			log.Printf("Warning: Error closing test file: %v", err)
		}
		if err := os.Remove(testFile); err != nil {
			log.Printf("Warning: Could not remove test file %s: %v", testFile, err)
		}
	}

	// Only log in debug mode to avoid polluting --help output
	if os.Getenv("ISH_DEBUG") != "" {
		log.Printf("Using database location: %s", xdgDBPath)
	}

	return xdgDBPath
}
