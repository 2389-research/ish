#!/usr/bin/env python3
# ABOUTME: Example script demonstrating Twilio API integration with ISH.
# ABOUTME: Shows how to send SMS, make calls, and manage phone numbers using the ISH fake Twilio API.

import requests
from typing import Optional
from datetime import datetime


class ISHTwilioClient:
    """Client for interacting with ISH's fake Twilio API."""

    def __init__(self, base_url: str = "http://localhost:9000",
                 account_sid: str = "AC_test_account", auth_token: str = "test_token"):
        self.base_url = base_url.rstrip("/")
        self.account_sid = account_sid
        self.auth_token = auth_token
        self.auth = (account_sid, auth_token)
        self.headers = {"Content-Type": "application/x-www-form-urlencoded"}

    def send_sms(self, to: str, from_: str, body: str):
        """Send an SMS message."""
        url = f"{self.base_url}/2010-04-01/Accounts/{self.account_sid}/Messages.json"
        data = {
            "To": to,
            "From": from_,
            "Body": body
        }
        response = requests.post(url, auth=self.auth, headers=self.headers, data=data)
        response.raise_for_status()
        return response.json()

    def get_message(self, message_sid: str):
        """Get details about a specific message."""
        url = f"{self.base_url}/2010-04-01/Accounts/{self.account_sid}/Messages/{message_sid}.json"
        response = requests.get(url, auth=self.auth)
        response.raise_for_status()
        return response.json()

    def list_messages(self, to: Optional[str] = None, from_: Optional[str] = None, limit: int = 20):
        """List sent/received messages."""
        url = f"{self.base_url}/2010-04-01/Accounts/{self.account_sid}/Messages.json"
        params = {"PageSize": limit}
        if to:
            params["To"] = to
        if from_:
            params["From"] = from_

        response = requests.get(url, auth=self.auth, params=params)
        response.raise_for_status()
        return response.json()

    def make_call(self, to: str, from_: str, url: str):
        """Initiate an outbound call."""
        endpoint = f"{self.base_url}/2010-04-01/Accounts/{self.account_sid}/Calls.json"
        data = {
            "To": to,
            "From": from_,
            "Url": url
        }
        response = requests.post(endpoint, auth=self.auth, headers=self.headers, data=data)
        response.raise_for_status()
        return response.json()

    def get_call(self, call_sid: str):
        """Get details about a specific call."""
        url = f"{self.base_url}/2010-04-01/Accounts/{self.account_sid}/Calls/{call_sid}.json"
        response = requests.get(url, auth=self.auth)
        response.raise_for_status()
        return response.json()

    def list_calls(self, to: Optional[str] = None, status: Optional[str] = None, limit: int = 20):
        """List calls."""
        url = f"{self.base_url}/2010-04-01/Accounts/{self.account_sid}/Calls.json"
        params = {"PageSize": limit}
        if to:
            params["To"] = to
        if status:
            params["Status"] = status

        response = requests.get(url, auth=self.auth, params=params)
        response.raise_for_status()
        return response.json()


def main():
    """Demonstrate Twilio API integration."""
    print("=" * 60)
    print("ISH Twilio API Integration Example")
    print("=" * 60)

    client = ISHTwilioClient()

    # 1. Send SMS messages
    print("\n1. Sending SMS messages:")
    print("-" * 60)
    try:
        # Send verification code
        msg1 = client.send_sms(
            to="+15555551234",
            from_="+15555559999",
            body="Your verification code is: 123456"
        )
        print(f"  Sent verification SMS")
        print(f"  SID: {msg1.get('sid', 'Unknown')}")
        print(f"  To: {msg1.get('to')}")
        print(f"  Status: {msg1.get('status')}")

        # Send notification
        msg2 = client.send_sms(
            to="+15555555678",
            from_="+15555559999",
            body="Your order #12345 has shipped! Track it at https://example.com/track"
        )
        print(f"\n  Sent notification SMS")
        print(f"  SID: {msg2.get('sid', 'Unknown')}")
        print(f"  Body: {msg2.get('body', '')[:50]}...")

    except Exception as e:
        print(f"  Error: {e}")

    # 2. List recent messages
    print("\n2. Listing recent messages:")
    print("-" * 60)
    try:
        messages = client.list_messages(limit=5)
        if "messages" in messages:
            print(f"  Found {len(messages['messages'])} messages")
            for msg in messages["messages"]:
                direction_arrow = "→" if msg.get("direction") == "outbound-api" else "←"
                print(f"  {direction_arrow} {msg.get('from')} to {msg.get('to')}")
                print(f"     {msg.get('body', 'No body')[:60]}...")
                print(f"     Status: {msg.get('status', 'unknown')}")
                print()
    except Exception as e:
        print(f"  Error: {e}")

    # 3. Get specific message
    print("\n3. Getting message details:")
    print("-" * 60)
    if "messages" in messages and messages["messages"]:
        try:
            msg_sid = messages["messages"][0]["sid"]
            detailed = client.get_message(msg_sid)
            print(f"  SID: {detailed.get('sid')}")
            print(f"  From: {detailed.get('from')}")
            print(f"  To: {detailed.get('to')}")
            print(f"  Body: {detailed.get('body')}")
            print(f"  Status: {detailed.get('status')}")
            print(f"  Date: {detailed.get('date_created')}")
            if "price" in detailed:
                print(f"  Price: {detailed.get('price')} {detailed.get('price_unit', 'USD')}")
        except Exception as e:
            print(f"  Error: {e}")

    # 4. Make phone calls
    print("\n4. Making outbound calls:")
    print("-" * 60)
    try:
        call1 = client.make_call(
            to="+15555551234",
            from_="+15555559999",
            url="https://demo.twilio.com/docs/voice.xml"
        )
        print(f"  Call initiated")
        print(f"  SID: {call1.get('sid', 'Unknown')}")
        print(f"  To: {call1.get('to')}")
        print(f"  From: {call1.get('from')}")
        print(f"  Status: {call1.get('status')}")
    except Exception as e:
        print(f"  Error: {e}")

    # 5. List calls
    print("\n5. Listing recent calls:")
    print("-" * 60)
    try:
        calls = client.list_calls(limit=5)
        if "calls" in calls:
            print(f"  Found {len(calls['calls'])} calls")
            for call in calls["calls"]:
                print(f"  {call.get('from')} → {call.get('to')}")
                print(f"    Status: {call.get('status', 'unknown')}")
                if "duration" in call:
                    print(f"    Duration: {call['duration']} seconds")
                print()
    except Exception as e:
        print(f"  Error: {e}")

    # 6. Send batch SMS (e.g., appointment reminders)
    print("\n6. Sending batch appointment reminders:")
    print("-" * 60)
    appointments = [
        ("+15555551111", "10:00 AM tomorrow"),
        ("+15555552222", "2:30 PM tomorrow"),
        ("+15555553333", "4:00 PM tomorrow")
    ]

    sent_count = 0
    for phone, time in appointments:
        try:
            msg = client.send_sms(
                to=phone,
                from_="+15555559999",
                body=f"Reminder: You have an appointment at {time}. Reply CONFIRM to confirm."
            )
            if msg.get("sid"):
                sent_count += 1
                print(f"  ✓ Sent to {phone} ({time})")
        except Exception as e:
            print(f"  ✗ Failed to send to {phone}: {e}")

    print(f"\n  Successfully sent {sent_count}/{len(appointments)} reminders")

    # 7. Send two-factor authentication code
    print("\n7. Sending 2FA code:")
    print("-" * 60)
    try:
        code = "987654"
        msg = client.send_sms(
            to="+15555551234",
            from_="+15555559999",
            body=f"Your login verification code is {code}. Valid for 5 minutes. Do not share this code."
        )
        print(f"  2FA code sent successfully")
        print(f"  Code: {code}")
        print(f"  To: {msg.get('to')}")
        print(f"  Message SID: {msg.get('sid')}")
    except Exception as e:
        print(f"  Error: {e}")

    # 8. Filter messages by recipient
    print("\n8. Filtering messages by recipient:")
    print("-" * 60)
    try:
        filtered = client.list_messages(to="+15555551234", limit=10)
        if "messages" in filtered:
            print(f"  Messages sent to +15555551234: {len(filtered['messages'])}")
            for msg in filtered["messages"][:3]:
                print(f"  - {msg.get('body', 'No body')[:50]}...")
    except Exception as e:
        print(f"  Error: {e}")

    print("\n" + "=" * 60)
    print("Example complete!")
    print("=" * 60)


if __name__ == "__main__":
    main()
