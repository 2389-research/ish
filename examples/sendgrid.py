#!/usr/bin/env python3
# ABOUTME: Example script demonstrating SendGrid API integration with ISH.
# ABOUTME: Shows how to send emails and manage suppression lists using the ISH fake SendGrid API.

import requests
from typing import Optional, List


class ISHSendGridClient:
    """Client for interacting with ISH's fake SendGrid API."""

    def __init__(self, base_url: str = "http://localhost:9000", api_key: Optional[str] = None):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key or "SG.test-api-key-from-ish"
        self.headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json"
        }

    def send_mail(self, to_email: str, from_email: str, subject: str,
                  content: str, content_type: str = "text/plain"):
        """Send an email via SendGrid API."""
        url = f"{self.base_url}/v3/mail/send"
        payload = {
            "personalizations": [
                {
                    "to": [{"email": to_email}]
                }
            ],
            "from": {"email": from_email},
            "subject": subject,
            "content": [
                {
                    "type": content_type,
                    "value": content
                }
            ]
        }
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.status_code == 202

    def get_suppressions(self, group_id: Optional[int] = None):
        """Get suppression list entries."""
        if group_id:
            url = f"{self.base_url}/v3/asm/groups/{group_id}/suppressions"
        else:
            url = f"{self.base_url}/v3/asm/suppressions"

        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()

    def add_suppression(self, emails: List[str], group_id: int = 1):
        """Add emails to suppression group."""
        url = f"{self.base_url}/v3/asm/groups/{group_id}/suppressions"
        payload = {"recipient_emails": emails}
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()

    def delete_suppression(self, email: str, group_id: int = 1):
        """Remove email from suppression group."""
        url = f"{self.base_url}/v3/asm/groups/{group_id}/suppressions/{email}"
        response = requests.delete(url, headers=self.headers)
        response.raise_for_status()
        return response.status_code == 204


def main():
    """Demonstrate SendGrid API integration."""
    print("=" * 60)
    print("ISH SendGrid API Integration Example")
    print("=" * 60)

    # Initialize client with ISH test API key
    client = ISHSendGridClient()

    # 1. Send a simple email
    print("\n1. Sending a simple email:")
    print("-" * 60)
    try:
        sent = client.send_mail(
            to_email="customer@example.com",
            from_email="noreply@myapp.com",
            subject="Welcome to Our Service!",
            content="Thank you for signing up. We're excited to have you!"
        )
        if sent:
            print("  Email sent successfully!")
            print("  To: customer@example.com")
            print("  From: noreply@myapp.com")
            print("  Subject: Welcome to Our Service!")
    except Exception as e:
        print(f"  Error: {e}")

    # 2. Send HTML email
    print("\n2. Sending HTML email:")
    print("-" * 60)
    try:
        html_content = """
        <html>
        <body>
            <h1>Password Reset</h1>
            <p>Click the link below to reset your password:</p>
            <a href="https://example.com/reset">Reset Password</a>
        </body>
        </html>
        """
        sent = client.send_mail(
            to_email="user@example.com",
            from_email="security@myapp.com",
            subject="Password Reset Request",
            content=html_content,
            content_type="text/html"
        )
        if sent:
            print("  HTML email sent successfully!")
            print("  To: user@example.com")
            print("  From: security@myapp.com")
    except Exception as e:
        print(f"  Error: {e}")

    # 3. Send transactional email
    print("\n3. Sending transactional email:")
    print("-" * 60)
    try:
        sent = client.send_mail(
            to_email="alice@example.com",
            from_email="billing@myapp.com",
            subject="Payment Received - Invoice #12345",
            content="""
Dear Alice,

We've received your payment of $99.00 for invoice #12345.

Transaction Details:
- Amount: $99.00
- Method: Credit Card ending in 4242
- Date: 2024-12-07

Thank you for your business!

Best regards,
The MyApp Team
            """.strip()
        )
        if sent:
            print("  Transaction email sent!")
            print("  To: alice@example.com")
            print("  Subject: Payment Received")
    except Exception as e:
        print(f"  Error: {e}")

    # 4. Get suppression list
    print("\n4. Checking suppression list:")
    print("-" * 60)
    try:
        suppressions = client.get_suppressions()
        if suppressions:
            print(f"  Found {len(suppressions)} suppressed emails:")
            for suppression in suppressions[:5]:
                print(f"  - {suppression.get('email', 'Unknown')}")
                if 'group_id' in suppression:
                    print(f"    Group: {suppression['group_id']}")
        else:
            print("  No suppressions found")
    except Exception as e:
        print(f"  Error: {e}")

    # 5. Add to suppression list
    print("\n5. Adding emails to suppression list:")
    print("-" * 60)
    try:
        result = client.add_suppression(
            emails=["bounced@example.com", "unsubscribed@example.com"],
            group_id=1
        )
        print("  Added emails to suppression group 1:")
        print("  - bounced@example.com")
        print("  - unsubscribed@example.com")
    except Exception as e:
        print(f"  Error: {e}")

    # 6. Try sending to suppressed email
    print("\n6. Attempting to send to suppressed email:")
    print("-" * 60)
    try:
        sent = client.send_mail(
            to_email="bounced@example.com",
            from_email="noreply@myapp.com",
            subject="This should be blocked",
            content="This email should not be delivered due to suppression"
        )
        print(f"  Send result: {sent}")
        print("  Note: In production, SendGrid would block this")
    except Exception as e:
        print(f"  Error (expected): {e}")

    # 7. Remove from suppression list
    print("\n7. Removing email from suppression list:")
    print("-" * 60)
    try:
        removed = client.delete_suppression("bounced@example.com", group_id=1)
        if removed:
            print("  Successfully removed bounced@example.com from suppressions")
    except Exception as e:
        print(f"  Error: {e}")

    # 8. Send batch emails
    print("\n8. Sending batch emails:")
    print("-" * 60)
    recipients = [
        "user1@example.com",
        "user2@example.com",
        "user3@example.com"
    ]

    sent_count = 0
    for recipient in recipients:
        try:
            sent = client.send_mail(
                to_email=recipient,
                from_email="newsletter@myapp.com",
                subject="Weekly Newsletter",
                content=f"Hello! This is your weekly update."
            )
            if sent:
                sent_count += 1
        except Exception as e:
            print(f"  Failed to send to {recipient}: {e}")

    print(f"  Successfully sent {sent_count}/{len(recipients)} emails")

    print("\n" + "=" * 60)
    print("Example complete!")
    print("=" * 60)


if __name__ == "__main__":
    main()
