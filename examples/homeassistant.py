#!/usr/bin/env python3
# ABOUTME: Example script demonstrating Home Assistant API integration with ISH.
# ABOUTME: Shows how to control smart home devices using the ISH fake Home Assistant API.

import requests
from typing import Optional, Dict, Any


class ISHHomeAssistantClient:
    """Client for interacting with ISH's fake Home Assistant API."""

    def __init__(self, base_url: str = "http://localhost:9000",
                 token: str = "token_home_main"):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.headers = {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json"
        }

    def get_states(self):
        """Get all entity states."""
        url = f"{self.base_url}/api/states"
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()

    def get_state(self, entity_id: str):
        """Get a specific entity state."""
        url = f"{self.base_url}/api/states/{entity_id}"
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()

    def set_state(self, entity_id: str, state: str, attributes: Optional[Dict[str, Any]] = None):
        """Set an entity state."""
        url = f"{self.base_url}/api/states/{entity_id}"
        payload = {
            "state": state
        }
        if attributes:
            payload["attributes"] = attributes

        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()

    def call_service(self, domain: str, service: str, entity_id: Optional[str] = None,
                    service_data: Optional[Dict[str, Any]] = None):
        """Call a Home Assistant service."""
        url = f"{self.base_url}/api/services/{domain}/{service}"
        payload = service_data or {}
        if entity_id:
            payload["entity_id"] = entity_id

        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()


def main():
    """Demonstrate Home Assistant API integration."""
    print("=" * 60)
    print("ISH Home Assistant API Integration Example")
    print("=" * 60)

    client = ISHHomeAssistantClient()

    # 1. Get all entity states
    print("\n1. Getting all entity states:")
    print("-" * 60)
    try:
        states = client.get_states()
        print(f"  Found {len(states)} entities")

        # Group by domain
        by_domain = {}
        for entity in states:
            domain = entity["entity_id"].split(".")[0]
            by_domain.setdefault(domain, []).append(entity)

        for domain, entities in sorted(by_domain.items()):
            print(f"\n  {domain.upper()}: {len(entities)} entities")
            for entity in entities[:3]:
                state_emoji = {
                    "on": "✅",
                    "off": "⭕",
                    "home": "🏠",
                    "away": "🚗"
                }.get(entity.get("state", "").lower(), "❓")
                print(f"    {state_emoji} {entity['entity_id']}: {entity.get('state')}")

    except Exception as e:
        print(f"  Error: {e}")

    # 2. Get specific entity state
    print("\n2. Getting specific entity states:")
    print("-" * 60)
    try:
        # Get light state
        light = client.get_state("light.living_room")
        print(f"  💡 Living Room Light: {light.get('state')}")
        if light.get("attributes"):
            attrs = light["attributes"]
            if "brightness" in attrs:
                print(f"     Brightness: {attrs['brightness']}/255")

        # Get temperature sensor
        temp = client.get_state("sensor.living_room_temperature")
        print(f"\n  🌡️  Living Room Temperature: {temp.get('state')}°F")
        if temp.get("attributes", {}).get("unit_of_measurement"):
            print(f"     Unit: {temp['attributes']['unit_of_measurement']}")

    except Exception as e:
        print(f"  Error: {e}")

    # 3. Turn on lights
    print("\n3. Turning on lights:")
    print("-" * 60)
    try:
        result = client.call_service(
            domain="light",
            service="turn_on",
            entity_id="light.living_room"
        )
        print(f"  ✅ Turned on living room light")

        # Turn on with brightness
        result = client.call_service(
            domain="light",
            service="turn_on",
            entity_id="light.bedroom",
            service_data={"brightness": 200}
        )
        print(f"  ✅ Turned on bedroom light at 78% brightness")

    except Exception as e:
        print(f"  Error: {e}")

    # 4. Control climate
    print("\n4. Controlling thermostat:")
    print("-" * 60)
    try:
        # Get current thermostat state
        thermo = client.get_state("climate.living_room")
        print(f"  Current temp: {thermo.get('state')}")
        print(f"  Target: {thermo.get('attributes', {}).get('temperature', 'N/A')}°F")

        # Set temperature
        result = client.call_service(
            domain="climate",
            service="set_temperature",
            entity_id="climate.living_room",
            service_data={"temperature": 72}
        )
        print(f"  ✅ Set thermostat to 72°F")

    except Exception as e:
        print(f"  Error: {e}")

    # 5. Control media player
    print("\n5. Controlling media player:")
    print("-" * 60)
    try:
        # Play media
        result = client.call_service(
            domain="media_player",
            service="play_media",
            entity_id="media_player.living_room_tv",
            service_data={
                "media_content_id": "spotify:playlist:37i9dQZF1DXcBWIGoYBM5M",
                "media_content_type": "playlist"
            }
        )
        print(f"  ▶️  Started playing music on living room TV")

        # Adjust volume
        result = client.call_service(
            domain="media_player",
            service="volume_set",
            entity_id="media_player.living_room_tv",
            service_data={"volume_level": 0.5}
        )
        print(f"  🔊 Set volume to 50%")

    except Exception as e:
        print(f"  Error: {e}")

    # 6. Set custom states
    print("\n6. Setting custom entity states:")
    print("-" * 60)
    try:
        result = client.set_state(
            entity_id="sensor.custom_counter",
            state="42",
            attributes={
                "unit_of_measurement": "items",
                "friendly_name": "Custom Counter"
            }
        )
        print(f"  ✅ Set custom counter to 42")

    except Exception as e:
        print(f"  Error: {e}")

    # 7. Create automation scenario
    print("\n7. Running automation scenario (Morning Routine):")
    print("-" * 60)
    try:
        print("  🌅 Morning routine starting...")

        # Turn on bedroom lights gradually
        client.call_service("light", "turn_on",
                          entity_id="light.bedroom",
                          service_data={"brightness": 100, "transition": 30})
        print("  ✓ Bedroom lights turning on gradually")

        # Set thermostat
        client.call_service("climate", "set_temperature",
                          entity_id="climate.bedroom",
                          service_data={"temperature": 70})
        print("  ✓ Thermostat set to 70°F")

        # Start coffee maker (switch)
        client.call_service("switch", "turn_on",
                          entity_id="switch.coffee_maker")
        print("  ✓ Coffee maker started")

        # Open blinds (cover)
        client.call_service("cover", "open_cover",
                          entity_id="cover.bedroom_blinds")
        print("  ✓ Blinds opening")

        print("\n  🎉 Morning routine complete!")

    except Exception as e:
        print(f"  Error: {e}")

    # 8. Check binary sensors
    print("\n8. Checking security sensors:")
    print("-" * 60)
    try:
        door = client.get_state("binary_sensor.front_door")
        motion = client.get_state("binary_sensor.living_room_motion")

        door_status = "🔓 Open" if door.get("state") == "on" else "🔒 Closed"
        motion_status = "🚶 Motion" if motion.get("state") == "on" else "✋ Clear"

        print(f"  Front Door: {door_status}")
        print(f"  Living Room Motion: {motion_status}")

    except Exception as e:
        print(f"  Error: {e}")

    # 9. Turn everything off (night mode)
    print("\n9. Activating night mode:")
    print("-" * 60)
    try:
        # Turn off all lights
        client.call_service("light", "turn_off")
        print("  ✅ All lights off")

        # Set thermostat to night mode
        client.call_service("climate", "set_temperature",
                          service_data={"temperature": 68})
        print("  ✅ Thermostat set to 68°F")

        # Ensure doors locked
        client.call_service("lock", "lock")
        print("  ✅ All doors locked")

        print("\n  🌙 Night mode activated")

    except Exception as e:
        print(f"  Error: {e}")

    print("\n" + "=" * 60)
    print("Example complete!")
    print("=" * 60)
    print("\nTip: Check seeded data for valid tokens:")
    print("  ./ish seed homeassistant")


if __name__ == "__main__":
    main()
