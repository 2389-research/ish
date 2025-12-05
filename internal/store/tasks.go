// ABOUTME: Tasks-related store operations for task lists and tasks.
// ABOUTME: Handles CRUD operations for Google Tasks API.

package store

import (
	"database/sql"
	"fmt"
	"time"
)

type TaskList struct {
	ID        string
	UserID    string
	Title     string
	UpdatedAt string
}

type Task struct {
	ID        string
	ListID    string
	Title     string
	Notes     string
	Due       string
	Status    string
	Completed string
	UpdatedAt string
}

// CreateTaskList creates a new task list
func (s *Store) CreateTaskList(tl *TaskList) error {
	if tl.ID == "" {
		tl.ID = fmt.Sprintf("tasklist_%d", time.Now().UnixNano())
	}
	tl.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		"INSERT INTO task_lists (id, user_id, title, updated_at) VALUES (?, ?, ?, ?)",
		tl.ID, tl.UserID, tl.Title, tl.UpdatedAt,
	)
	return err
}

// GetTaskList retrieves a task list by ID
func (s *Store) GetTaskList(listID string) (*TaskList, error) {
	var tl TaskList
	err := s.db.QueryRow(
		"SELECT id, user_id, title, COALESCE(updated_at, '') FROM task_lists WHERE id = ?",
		listID,
	).Scan(&tl.ID, &tl.UserID, &tl.Title, &tl.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task list not found")
	}
	return &tl, err
}

// CreateTask creates a new task
func (s *Store) CreateTask(t *Task) (*Task, error) {
	if t.ID == "" {
		t.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		`INSERT INTO tasks (id, list_id, title, notes, due, status, completed, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ListID, t.Title, t.Notes, t.Due, t.Status, t.Completed, t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetTask retrieves a task by list ID and task ID
func (s *Store) GetTask(listID, taskID string) (*Task, error) {
	var t Task
	err := s.db.QueryRow(
		`SELECT id, list_id, title, COALESCE(notes, ''), COALESCE(due, ''), status,
		 COALESCE(completed, ''), COALESCE(updated_at, '') FROM tasks
		 WHERE list_id = ? AND id = ?`,
		listID, taskID,
	).Scan(&t.ID, &t.ListID, &t.Title, &t.Notes, &t.Due, &t.Status, &t.Completed, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	return &t, err
}

// ListTasks lists tasks in a task list
func (s *Store) ListTasks(listID string, showCompleted bool, maxResults int64) ([]*Task, error) {
	query := `SELECT id, list_id, title, COALESCE(notes, ''), COALESCE(due, ''), status,
			  COALESCE(completed, ''), COALESCE(updated_at, '') FROM tasks
			  WHERE list_id = ?`
	args := []any{listID}

	if !showCompleted {
		query += " AND status != 'completed'"
	}

	query += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, maxResults)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ListID, &t.Title, &t.Notes, &t.Due, &t.Status, &t.Completed, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, nil
}

// UpdateTask updates an existing task
func (s *Store) UpdateTask(t *Task) (*Task, error) {
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		`UPDATE tasks SET title = ?, notes = ?, due = ?, status = ?, completed = ?, updated_at = ?
		 WHERE list_id = ? AND id = ?`,
		t.Title, t.Notes, t.Due, t.Status, t.Completed, t.UpdatedAt, t.ListID, t.ID,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTask deletes a task
func (s *Store) DeleteTask(listID, taskID string) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE list_id = ? AND id = ?", listID, taskID)
	return err
}

// ListAllTasks lists all tasks for admin UI
func (s *Store) ListAllTasks() ([]*Task, error) {
	query := `SELECT id, list_id, title, COALESCE(notes, ''), COALESCE(due, ''), status,
			  COALESCE(completed, ''), COALESCE(updated_at, '') FROM tasks
			  ORDER BY updated_at DESC LIMIT 100`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ListID, &t.Title, &t.Notes, &t.Due, &t.Status, &t.Completed, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, nil
}

// CreateTaskFromForm creates a task from admin form input
func (s *Store) CreateTaskFromForm(title, notes, due, status string) (*Task, error) {
	task := &Task{
		ListID: "@default",
		Title:  title,
		Notes:  notes,
		Due:    due,
		Status: status,
	}
	return s.CreateTask(task)
}
