#!/usr/bin/env python3
# ABOUTME: Example script demonstrating Calendar API integration with ISH.
# ABOUTME: Shows how to list, create, update, and delete calendar events.

import requests
from typing import Optional
from datetime import datetime, timedelta


class ISHCalendarClient:
    """Client for interacting with ISH's fake Google Calendar API."""

    def __init__(self, base_url: str = "http://localhost:9000", calendar_id: str = "primary"):
        self.base_url = base_url.rstrip("/")
        self.calendar_id = calendar_id
        self.headers = {
            "Authorization": "Bearer user:me",
            "Content-Type": "application/json"
        }

    def list_events(self, time_min: Optional[str] = None, time_max: Optional[str] = None, max_results: int = 10):
        """List events on the calendar."""
        params = {"maxResults": max_results}
        if time_min:
            params["timeMin"] = time_min
        if time_max:
            params["timeMax"] = time_max

        url = f"{self.base_url}/calendar/v3/calendars/{self.calendar_id}/events"
        response = requests.get(url, headers=self.headers, params=params)
        response.raise_for_status()
        return response.json()

    def get_event(self, event_id: str):
        """Get a specific event by ID."""
        url = f"{self.base_url}/calendar/v3/calendars/{self.calendar_id}/events/{event_id}"
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()

    def create_event(self, summary: str, start: str, end: str, description: str = "", location: str = ""):
        """Create a new calendar event."""
        url = f"{self.base_url}/calendar/v3/calendars/{self.calendar_id}/events"
        payload = {
            "summary": summary,
            "description": description,
            "location": location,
            "start": {"dateTime": start},
            "end": {"dateTime": end}
        }
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()

    def update_event(self, event_id: str, **updates):
        """Update an existing event."""
        url = f"{self.base_url}/calendar/v3/calendars/{self.calendar_id}/events/{event_id}"
        response = requests.put(url, headers=self.headers, json=updates)
        response.raise_for_status()
        return response.json()

    def delete_event(self, event_id: str):
        """Delete an event."""
        url = f"{self.base_url}/calendar/v3/calendars/{self.calendar_id}/events/{event_id}"
        response = requests.delete(url, headers=self.headers)
        response.raise_for_status()
        return response.status_code == 204


def main():
    """Demonstrate Calendar API integration."""
    print("=" * 60)
    print("ISH Calendar API Integration Example")
    print("=" * 60)

    client = ISHCalendarClient()

    # 1. List upcoming events
    print("\n1. Listing upcoming events:")
    print("-" * 60)
    now = datetime.utcnow().isoformat() + "Z"
    events = client.list_events(time_min=now, max_results=5)
    if "items" in events:
        print(f"  Found {len(events['items'])} events")
        for event in events["items"]:
            start = event.get("start", {}).get("dateTime", "No start time")
            print(f"  - {event.get('summary', 'No title')}")
            print(f"    Start: {start}")
            print(f"    Location: {event.get('location', 'No location')}")
            print()
    else:
        print("  No events found")

    # 2. Get specific event details
    print("\n2. Getting event details:")
    print("-" * 60)
    if "items" in events and events["items"]:
        event_id = events["items"][0]["id"]
        detailed = client.get_event(event_id)
        print(f"  ID: {detailed['id']}")
        print(f"  Summary: {detailed.get('summary', 'No title')}")
        print(f"  Description: {detailed.get('description', 'No description')}")
        print(f"  Location: {detailed.get('location', 'No location')}")
        print(f"  Start: {detailed.get('start', {}).get('dateTime', 'Unknown')}")
        print(f"  End: {detailed.get('end', {}).get('dateTime', 'Unknown')}")

    # 3. Create a new event
    print("\n3. Creating a new event:")
    print("-" * 60)
    tomorrow = datetime.utcnow() + timedelta(days=1)
    start_time = tomorrow.replace(hour=14, minute=0, second=0, microsecond=0)
    end_time = start_time + timedelta(hours=1)

    try:
        new_event = client.create_event(
            summary="Team Planning Session",
            description="Quarterly planning and roadmap review",
            location="Conference Room A",
            start=start_time.isoformat() + "Z",
            end=end_time.isoformat() + "Z"
        )
        print(f"  Event created! ID: {new_event.get('id', 'Unknown')}")
        print(f"  Summary: {new_event.get('summary')}")
        print(f"  Start: {new_event.get('start', {}).get('dateTime')}")

        # 4. Update the event
        print("\n4. Updating the event:")
        print("-" * 60)
        updated = client.update_event(
            new_event["id"],
            summary="Q1 Team Planning Session (Updated)",
            location="Conference Room B"
        )
        print(f"  Updated event: {updated.get('summary')}")
        print(f"  New location: {updated.get('location')}")

        # 5. Delete the event
        print("\n5. Deleting the event:")
        print("-" * 60)
        deleted = client.delete_event(new_event["id"])
        if deleted:
            print(f"  Event {new_event['id']} deleted successfully")

    except Exception as e:
        print(f"  Error: {e}")

    # 6. List all events (no filters)
    print("\n6. Listing all events:")
    print("-" * 60)
    all_events = client.list_events(max_results=10)
    if "items" in all_events:
        print(f"  Total events: {len(all_events['items'])}")
        for event in all_events["items"][:5]:
            print(f"  - {event.get('summary', 'No title')}: {event.get('start', {}).get('dateTime', 'No time')}")

    print("\n" + "=" * 60)
    print("Example complete!")
    print("=" * 60)


if __name__ == "__main__":
    main()
