// ABOUTME: Entry point for the ISH fake Google API server.
// ABOUTME: Wires together store, auth, and API handlers with CLI commands.

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/cobra"
	"github.com/2389/ish/internal/admin"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/logging"
	"github.com/2389/ish/internal/seed"
	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	_ "github.com/2389/ish/plugins/google" // Register Google plugin
	_ "github.com/2389/ish/plugins/oauth"  // Register OAuth plugin
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

	// Initialize all plugins with store
	for _, plugin := range core.All() {
		// Set store for plugins that need it
		type storePlugin interface {
			SetStore(*store.Store)
		}
		if sp, ok := plugin.(storePlugin); ok {
			sp.SetStore(s)
		}
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
	userID := "harper"

	// Create default user
	if err := s.CreateUser(userID); err != nil {
		return err
	}
	log.Println("Created user:", userID)

	// Generate data using AI or static fallback
	gen := seed.NewGenerator(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	data, err := gen.Generate(ctx, 50, 25, 25)
	if err != nil {
		return fmt.Errorf("failed to generate seed data: %w", err)
	}

	// Insert Gmail data
	baseTime := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
	for i, email := range data.Emails {
		threadID := fmt.Sprintf("thr_%d", i+1)
		msgID := fmt.Sprintf("msg_%d", i+1)

		// Create thread
		snippet := email.Body
		if len(snippet) > 100 {
			snippet = snippet[:100] + "..."
		}
		s.CreateGmailThread(&store.GmailThread{
			ID:      threadID,
			UserID:  userID,
			Snippet: snippet,
		})

		// Build payload with headers and body
		bodyEncoded := base64.StdEncoding.EncodeToString([]byte(email.Body))
		payload := fmt.Sprintf(`{"headers":[{"name":"From","value":"%s"},{"name":"To","value":"%s"},{"name":"Subject","value":"%s"}],"body":{"data":"%s"}}`,
			email.From, email.To, email.Subject, bodyEncoded)

		s.CreateGmailMessage(&store.GmailMessage{
			ID:           msgID,
			UserID:       userID,
			ThreadID:     threadID,
			LabelIDs:     email.Labels,
			Snippet:      snippet,
			InternalDate: baseTime + int64(i*60000), // 1 min apart
			Payload:      payload,
		})
	}
	log.Printf("Created %d Gmail messages", len(data.Emails))

	// Insert Calendar data
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: userID, Summary: "Primary Calendar"})

	for i, event := range data.Events {
		attendeesJSON, _ := json.Marshal(func() []map[string]string {
			result := make([]map[string]string, len(event.Attendees))
			for j, email := range event.Attendees {
				result[j] = map[string]string{"email": email}
			}
			return result
		}())

		s.CreateCalendarEvent(&store.CalendarEvent{
			ID:          fmt.Sprintf("evt_%d", i+1),
			CalendarID:  "primary",
			Summary:     event.Summary,
			Description: event.Description,
			StartTime:   event.StartTime,
			EndTime:     event.EndTime,
			Attendees:   string(attendeesJSON),
		})
	}
	log.Printf("Created %d Calendar events", len(data.Events))

	// Insert People data
	for i, contact := range data.Contacts {
		personData := map[string]any{
			"names": []map[string]string{
				{"displayName": contact.Name},
			},
			"emailAddresses": []map[string]string{
				{"value": contact.Email},
			},
		}
		if contact.Phone != "" {
			personData["phoneNumbers"] = []map[string]string{
				{"value": contact.Phone},
			}
		}
		if contact.Company != "" {
			personData["organizations"] = []map[string]string{
				{"name": contact.Company},
			}
		}
		dataJSON, _ := json.Marshal(personData)

		s.CreatePerson(&store.Person{
			ResourceName: fmt.Sprintf("people/c%d", i+1),
			UserID:       userID,
			Data:         string(dataJSON),
		})
	}
	log.Printf("Created %d People contacts", len(data.Contacts))

	// Create default task list
	s.CreateTaskList(&store.TaskList{
		ID:     "@default",
		UserID: userID,
		Title:  "My Tasks",
	})
	log.Println("Created default task list")

	// Create sample tasks
	taskData := []struct {
		title  string
		notes  string
		due    string
		status string
	}{
		{"Review project proposal", "High priority - needs review by end of week", time.Now().Add(3 * 24 * time.Hour).Format(time.RFC3339), "needsAction"},
		{"Prepare quarterly report", "Include metrics from last quarter", time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339), "needsAction"},
		{"Schedule team meeting", "Discuss upcoming sprint planning", time.Now().Add(2 * 24 * time.Hour).Format(time.RFC3339), "needsAction"},
		{"Update documentation", "Update API documentation with new endpoints", time.Now().Add(5 * 24 * time.Hour).Format(time.RFC3339), "needsAction"},
		{"Fix production bug", "User reported login issues", time.Now().Add(1 * 24 * time.Hour).Format(time.RFC3339), "needsAction"},
		{"Refactor authentication logic", "Improve code maintainability", time.Now().Add(10 * 24 * time.Hour).Format(time.RFC3339), "needsAction"},
		{"Write unit tests", "Increase test coverage to 80%", time.Now().Add(14 * 24 * time.Hour).Format(time.RFC3339), "needsAction"},
		{"Deploy to staging environment", "Test all new features before production deploy", time.Now().Add(4 * 24 * time.Hour).Format(time.RFC3339), "needsAction"},
		{"Review pull requests", "Check pending PRs from team", time.Now().Add(-1 * 24 * time.Hour).Format(time.RFC3339), "completed"},
		{"Set up CI/CD pipeline", "Automated testing and deployment", time.Now().Add(-3 * 24 * time.Hour).Format(time.RFC3339), "completed"},
	}

	for i, td := range taskData {
		task := &store.Task{
			ID:      fmt.Sprintf("task_%d", i+1),
			ListID:  "@default",
			Title:   td.title,
			Notes:   td.notes,
			Due:     td.due,
			Status:  td.status,
			UpdatedAt: time.Now().Add(-time.Duration(len(taskData)-i) * time.Hour).Format(time.RFC3339),
		}
		if td.status == "completed" {
			task.Completed = time.Now().Add(-time.Duration(i) * 24 * time.Hour).Format(time.RFC3339)
		}
		s.CreateTask(task)
	}
	log.Printf("Created %d tasks", len(taskData))

	log.Println("Seed complete!")
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
