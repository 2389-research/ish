#!/usr/bin/env python3
# ABOUTME: Example script demonstrating Gmail API integration with ISH.
# ABOUTME: Shows how to list, search, and send emails using the ISH fake Gmail API.

import requests
from typing import Optional
from datetime import datetime


class ISHGmailClient:
    """Client for interacting with ISH's fake Gmail API."""

    def __init__(self, base_url: str = "http://localhost:9000", user_id: str = "me"):
        self.base_url = base_url.rstrip("/")
        self.user_id = user_id
        self.headers = {
            "Authorization": f"Bearer user:{user_id}",
            "Content-Type": "application/json"
        }

    def list_messages(self, max_results: int = 10, q: Optional[str] = None):
        """List messages in the user's mailbox."""
        params = {"maxResults": max_results}
        if q:
            params["q"] = q

        url = f"{self.base_url}/gmail/v1/users/{self.user_id}/messages"
        response = requests.get(url, headers=self.headers, params=params)
        response.raise_for_status()
        return response.json()

    def get_message(self, message_id: str):
        """Get a specific message by ID."""
        url = f"{self.base_url}/gmail/v1/users/{self.user_id}/messages/{message_id}"
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()

    def send_message(self, to: str, subject: str, body: str):
        """Send an email message."""
        url = f"{self.base_url}/gmail/v1/users/{self.user_id}/messages/send"
        payload = {
            "raw": {
                "to": to,
                "subject": subject,
                "body": body
            }
        }
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()

    def trash_message(self, message_id: str):
        """Move a message to trash."""
        url = f"{self.base_url}/gmail/v1/users/{self.user_id}/messages/{message_id}/trash"
        response = requests.post(url, headers=self.headers)
        response.raise_for_status()
        return response.json()


def main():
    """Demonstrate Gmail API integration."""
    print("=" * 60)
    print("ISH Gmail API Integration Example")
    print("=" * 60)

    client = ISHGmailClient()

    # 1. List all messages
    print("\n1. Listing all messages:")
    print("-" * 60)
    messages = client.list_messages(max_results=5)
    if "messages" in messages:
        for msg in messages["messages"]:
            print(f"  ID: {msg['id']}")
            print(f"  ThreadID: {msg['threadId']}")
            if "snippet" in msg:
                print(f"  Snippet: {msg['snippet'][:80]}...")
            print()
    else:
        print("  No messages found")

    # 2. Search for specific messages
    print("\n2. Searching for messages with 'team' in subject:")
    print("-" * 60)
    search_results = client.list_messages(q="subject:team")
    if "messages" in search_results:
        print(f"  Found {len(search_results['messages'])} messages")
        for msg in search_results["messages"][:3]:
            full_msg = client.get_message(msg["id"])
            print(f"  - {full_msg.get('snippet', 'No snippet')[:60]}...")
    else:
        print("  No matching messages found")

    # 3. Get detailed message
    print("\n3. Getting detailed message:")
    print("-" * 60)
    if "messages" in messages and messages["messages"]:
        msg_id = messages["messages"][0]["id"]
        detailed = client.get_message(msg_id)
        print(f"  ID: {detailed['id']}")
        print(f"  Snippet: {detailed.get('snippet', 'No snippet')}")
        print(f"  Labels: {', '.join(detailed.get('labelIds', []))}")
        if "payload" in detailed and "headers" in detailed["payload"]:
            headers = {h["name"]: h["value"] for h in detailed["payload"]["headers"]}
            print(f"  From: {headers.get('From', 'Unknown')}")
            print(f"  Subject: {headers.get('Subject', 'No subject')}")
            print(f"  Date: {headers.get('Date', 'Unknown')}")

    # 4. Send a new message
    print("\n4. Sending a new message:")
    print("-" * 60)
    try:
        sent = client.send_message(
            to="colleague@example.com",
            subject="Test from ISH Client",
            body="This is a test message sent via ISH's Gmail API!"
        )
        print(f"  Message sent! ID: {sent.get('id', 'Unknown')}")
        print(f"  Labels: {', '.join(sent.get('labelIds', []))}")
    except Exception as e:
        print(f"  Error sending message: {e}")

    # 5. Demonstrate error handling
    print("\n5. Error handling example:")
    print("-" * 60)
    try:
        client.get_message("nonexistent-message-id")
    except requests.exceptions.HTTPError as e:
        print(f"  Caught expected error: {e.response.status_code}")
        print(f"  Error message: {e.response.text[:100]}")

    print("\n" + "=" * 60)
    print("Example complete!")
    print("=" * 60)


if __name__ == "__main__":
    main()
