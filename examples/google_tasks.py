#!/usr/bin/env python3
# ABOUTME: Example script demonstrating Tasks API integration with ISH.
# ABOUTME: Shows how to manage tasks and task lists using the ISH fake Tasks API.

import requests
from typing import Optional
from datetime import datetime, timedelta


class ISHTasksClient:
    """Client for interacting with ISH's fake Google Tasks API."""

    def __init__(self, base_url: str = "http://localhost:9000", tasklist: str = "@default"):
        self.base_url = base_url.rstrip("/")
        self.tasklist = tasklist
        self.headers = {
            "Authorization": "Bearer user:me",
            "Content-Type": "application/json"
        }

    def list_tasks(self, show_completed: bool = False, show_hidden: bool = False):
        """List tasks in the task list."""
        params = {
            "showCompleted": str(show_completed).lower(),
            "showHidden": str(show_hidden).lower()
        }
        url = f"{self.base_url}/tasks/v1/lists/{self.tasklist}/tasks"
        response = requests.get(url, headers=self.headers, params=params)
        response.raise_for_status()
        return response.json()

    def get_task(self, task_id: str):
        """Get a specific task by ID."""
        url = f"{self.base_url}/tasks/v1/lists/{self.tasklist}/tasks/{task_id}"
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()

    def create_task(self, title: str, notes: str = "", due: Optional[str] = None):
        """Create a new task."""
        url = f"{self.base_url}/tasks/v1/lists/{self.tasklist}/tasks"
        payload = {
            "title": title,
            "notes": notes
        }
        if due:
            payload["due"] = due

        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()

    def update_task(self, task_id: str, **updates):
        """Update an existing task."""
        url = f"{self.base_url}/tasks/v1/lists/{self.tasklist}/tasks/{task_id}"
        response = requests.put(url, headers=self.headers, json=updates)
        response.raise_for_status()
        return response.json()

    def complete_task(self, task_id: str):
        """Mark a task as completed."""
        return self.update_task(task_id, status="completed")

    def delete_task(self, task_id: str):
        """Delete a task."""
        url = f"{self.base_url}/tasks/v1/lists/{self.tasklist}/tasks/{task_id}"
        response = requests.delete(url, headers=self.headers)
        response.raise_for_status()
        return response.status_code == 204


def main():
    """Demonstrate Tasks API integration."""
    print("=" * 60)
    print("ISH Tasks API Integration Example")
    print("=" * 60)

    client = ISHTasksClient()

    # 1. List all tasks
    print("\n1. Listing all tasks:")
    print("-" * 60)
    tasks = client.list_tasks()
    if "items" in tasks:
        print(f"  Found {len(tasks['items'])} tasks")
        for task in tasks["items"]:
            status_emoji = "✅" if task.get("status") == "completed" else "⬜"
            print(f"  {status_emoji} {task.get('title', 'Untitled')}")
            if task.get("notes"):
                print(f"      Notes: {task['notes'][:60]}...")
            if task.get("due"):
                print(f"      Due: {task['due']}")
            print()
    else:
        print("  No tasks found")

    # 2. Get specific task details
    print("\n2. Getting task details:")
    print("-" * 60)
    if "items" in tasks and tasks["items"]:
        task_id = tasks["items"][0]["id"]
        detailed = client.get_task(task_id)
        print(f"  ID: {detailed['id']}")
        print(f"  Title: {detailed.get('title', 'Untitled')}")
        print(f"  Status: {detailed.get('status', 'Unknown')}")
        print(f"  Notes: {detailed.get('notes', 'No notes')}")
        print(f"  Due: {detailed.get('due', 'No due date')}")
        print(f"  Updated: {detailed.get('updated', 'Unknown')}")

    # 3. Create new tasks
    print("\n3. Creating new tasks:")
    print("-" * 60)

    # Create a task with a due date
    tomorrow = (datetime.utcnow() + timedelta(days=1)).isoformat() + "Z"
    try:
        task1 = client.create_task(
            title="Write integration tests",
            notes="Add comprehensive tests for the new API endpoints",
            due=tomorrow
        )
        print(f"  Created task: {task1.get('title')}")
        print(f"  ID: {task1.get('id')}")
        print(f"  Due: {task1.get('due')}")

        # Create a task without a due date
        task2 = client.create_task(
            title="Review documentation",
            notes="Check all docs are up to date with latest changes"
        )
        print(f"\n  Created task: {task2.get('title')}")
        print(f"  ID: {task2.get('id')}")

        # 4. Update a task
        print("\n4. Updating task:")
        print("-" * 60)
        updated = client.update_task(
            task1["id"],
            title="Write integration and unit tests",
            notes="Add comprehensive tests for the new API endpoints (updated)"
        )
        print(f"  Updated title: {updated.get('title')}")
        print(f"  Updated notes: {updated.get('notes')}")

        # 5. Complete a task
        print("\n5. Completing task:")
        print("-" * 60)
        completed = client.complete_task(task2["id"])
        print(f"  Task '{completed.get('title')}' marked as: {completed.get('status')}")

        # 6. List completed tasks
        print("\n6. Listing completed tasks:")
        print("-" * 60)
        completed_tasks = client.list_tasks(show_completed=True)
        if "items" in completed_tasks:
            completed_count = sum(1 for t in completed_tasks["items"] if t.get("status") == "completed")
            print(f"  Total completed tasks: {completed_count}")
            for task in [t for t in completed_tasks["items"] if t.get("status") == "completed"][:3]:
                print(f"  ✅ {task.get('title')}")

        # 7. Delete a task
        print("\n7. Deleting task:")
        print("-" * 60)
        deleted = client.delete_task(task1["id"])
        if deleted:
            print(f"  Task {task1['id']} deleted successfully")

    except Exception as e:
        print(f"  Error: {e}")

    # 8. Final task count
    print("\n8. Final task summary:")
    print("-" * 60)
    all_tasks = client.list_tasks(show_completed=True)
    if "items" in all_tasks:
        total = len(all_tasks["items"])
        completed = sum(1 for t in all_tasks["items"] if t.get("status") == "completed")
        pending = total - completed
        print(f"  Total tasks: {total}")
        print(f"  Completed: {completed}")
        print(f"  Pending: {pending}")

    print("\n" + "=" * 60)
    print("Example complete!")
    print("=" * 60)


if __name__ == "__main__":
    main()
