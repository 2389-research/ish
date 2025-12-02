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
	"github.com/2389/ish/internal/calendar"
	"github.com/2389/ish/internal/gmail"
	"github.com/2389/ish/internal/people"
	"github.com/2389/ish/internal/store"
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
	r.Use(auth.Middleware)

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	// API handlers
	gmail.NewHandlers(s).RegisterRoutes(r)
	calendar.NewHandlers(s).RegisterRoutes(r)
	people.NewHandlers(s).RegisterRoutes(r)
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
	// Create default user
	if err := s.CreateUser("harper"); err != nil {
		return err
	}
	log.Println("Created user: harper")

	// Gmail data
	s.CreateGmailThread(&store.GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Welcome to ISH"})
	s.CreateGmailThread(&store.GmailThread{ID: "thr_2", UserID: "harper", Snippet: "Meeting tomorrow"})

	messages := []store.GmailMessage{
		{ID: "msg_1", UserID: "harper", ThreadID: "thr_1", LabelIDs: []string{"INBOX"}, Snippet: "Welcome to ISH, your fake Google API server!", InternalDate: 1733000000000, Payload: `{"headers":[{"name":"From","value":"ish@example.com"},{"name":"Subject","value":"Welcome to ISH"}]}`},
		{ID: "msg_2", UserID: "harper", ThreadID: "thr_1", LabelIDs: []string{"INBOX"}, Snippet: "Getting started guide attached.", InternalDate: 1733000100000, Payload: `{"headers":[{"name":"From","value":"ish@example.com"},{"name":"Subject","value":"Re: Welcome to ISH"}]}`},
		{ID: "msg_3", UserID: "harper", ThreadID: "thr_2", LabelIDs: []string{"INBOX", "STARRED"}, Snippet: "Don't forget our meeting tomorrow at 10am.", InternalDate: 1733000200000, Payload: `{"headers":[{"name":"From","value":"alice@example.com"},{"name":"Subject","value":"Meeting tomorrow"}]}`},
		{ID: "msg_4", UserID: "harper", ThreadID: "thr_2", LabelIDs: []string{"INBOX"}, Snippet: "I'll bring the coffee!", InternalDate: 1733000300000, Payload: `{"headers":[{"name":"From","value":"bob@example.com"},{"name":"Subject","value":"Re: Meeting tomorrow"}]}`},
		{ID: "msg_5", UserID: "harper", ThreadID: "thr_2", LabelIDs: []string{"INBOX", "IMPORTANT"}, Snippet: "Agenda attached for review.", InternalDate: 1733000400000, Payload: `{"headers":[{"name":"From","value":"alice@example.com"},{"name":"Subject","value":"Re: Meeting tomorrow"}]}`},
	}
	for _, m := range messages {
		s.CreateGmailMessage(&m)
	}
	log.Printf("Created %d Gmail messages", len(messages))

	// Calendar data
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: "harper", Summary: "Primary Calendar"})

	events := []store.CalendarEvent{
		{ID: "evt_1", CalendarID: "primary", Summary: "Team Standup", Description: "Daily sync", StartTime: "2025-12-01T09:00:00Z", EndTime: "2025-12-01T09:30:00Z", Attendees: `[{"email":"harper@example.com"},{"email":"alice@example.com"}]`},
		{ID: "evt_2", CalendarID: "primary", Summary: "Project Review", Description: "Q4 review", StartTime: "2025-12-01T14:00:00Z", EndTime: "2025-12-01T15:00:00Z", Attendees: `[{"email":"harper@example.com"},{"email":"bob@example.com"}]`},
		{ID: "evt_3", CalendarID: "primary", Summary: "Coffee Chat", Description: "Casual sync", StartTime: "2025-12-02T10:00:00Z", EndTime: "2025-12-02T10:30:00Z", Attendees: `[{"email":"harper@example.com"}]`},
	}
	for _, e := range events {
		s.CreateCalendarEvent(&e)
	}
	log.Printf("Created %d Calendar events", len(events))

	// People data
	contacts := []store.Person{
		{ResourceName: "people/c1", UserID: "harper", Data: `{"names":[{"displayName":"Alice Smith"}],"emailAddresses":[{"value":"alice@example.com"}],"photos":[{"url":"https://example.com/alice.png"}]}`},
		{ResourceName: "people/c2", UserID: "harper", Data: `{"names":[{"displayName":"Bob Jones"}],"emailAddresses":[{"value":"bob@example.com"}],"photos":[{"url":"https://example.com/bob.png"}]}`},
		{ResourceName: "people/c3", UserID: "harper", Data: `{"names":[{"displayName":"Charlie Brown"}],"emailAddresses":[{"value":"charlie@example.com"}],"photos":[{"url":"https://example.com/charlie.png"}]}`},
	}
	for _, p := range contacts {
		s.CreatePerson(&p)
	}
	log.Printf("Created %d People contacts", len(contacts))

	log.Println("Seed complete!")
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
