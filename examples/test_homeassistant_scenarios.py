#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "requests",
#     "websockets",
# ]
# ///

# ABOUTME: Scenario-based integration tests for Home Assistant API
# ABOUTME: Tests real-world workflows like morning routines, device control, and automation

import asyncio
import websockets
import json
import requests
from typing import Dict, Any, List


class HomeAssistantScenarioTester:
    """Test Home Assistant integration with real-world scenarios"""

    def __init__(self, base_url: str = "http://localhost:9000", token: str = "token_home_main"):
        self.base_url = base_url
        self.token = token
        self.headers = {"Authorization": f"Bearer {token}"}
        self.ws_url = base_url.replace("http://", "ws://").replace("https://", "wss://") + "/api/websocket"

    def print_scenario(self, name: str):
        """Print scenario header"""
        print("\n" + "=" * 70)
        print(f"SCENARIO: {name}")
        print("=" * 70)

    def print_step(self, step: str):
        """Print test step"""
        print(f"\n→ {step}")

    def print_success(self, message: str):
        """Print success message"""
        print(f"  ✓ {message}")

    def print_failure(self, message: str):
        """Print failure message"""
        print(f"  ✗ {message}")

    def get_states(self) -> List[Dict[str, Any]]:
        """Get all entity states via REST API"""
        response = requests.get(f"{self.base_url}/api/states", headers=self.headers)
        response.raise_for_status()
        return response.json()

    def get_state(self, entity_id: str) -> Dict[str, Any]:
        """Get single entity state"""
        response = requests.get(f"{self.base_url}/api/states/{entity_id}", headers=self.headers)
        response.raise_for_status()
        return response.json()

    def set_state(self, entity_id: str, state: str, attributes: Dict[str, Any] = None) -> Dict[str, Any]:
        """Set entity state"""
        payload = {"state": state}
        if attributes:
            payload["attributes"] = attributes

        response = requests.post(
            f"{self.base_url}/api/states/{entity_id}",
            headers=self.headers,
            json=payload
        )
        response.raise_for_status()
        return response.json()

    def call_service(self, domain: str, service: str, entity_id: str = None, **kwargs) -> List[Dict[str, Any]]:
        """Call a service"""
        payload = {}
        if entity_id:
            payload["entity_id"] = entity_id
        payload.update(kwargs)

        response = requests.post(
            f"{self.base_url}/api/services/{domain}/{service}",
            headers=self.headers,
            json=payload
        )
        response.raise_for_status()
        return response.json()

    async def websocket_scenario(self):
        """Test WebSocket real-time updates scenario"""
        self.print_scenario("Real-time Device Monitoring via WebSocket")

        async with websockets.connect(self.ws_url) as ws:
            self.print_step("Connecting to WebSocket")

            # Auth flow
            msg = await ws.recv()
            auth_req = json.loads(msg)
            self.print_success(f"Received {auth_req['type']}")

            await ws.send(json.dumps({"type": "auth", "access_token": self.token}))
            msg = await ws.recv()
            auth_ok = json.loads(msg)

            if auth_ok["type"] == "auth_ok":
                self.print_success("WebSocket authenticated")
            else:
                self.print_failure(f"Auth failed: {auth_ok}")
                return

            # Get current states
            self.print_step("Requesting all device states")
            await ws.send(json.dumps({"id": 1, "type": "get_states"}))
            msg = await ws.recv()
            states_response = json.loads(msg)

            if states_response.get("success"):
                states = states_response.get("result", [])
                self.print_success(f"Retrieved {len(states)} device states")

                # Show sample devices
                for state in states[:3]:
                    entity_id = state.get("entity_id", "unknown")
                    current_state = state.get("state", "unknown")
                    print(f"    • {entity_id}: {current_state}")
            else:
                self.print_failure("Failed to get states")

            # Test ping/pong
            self.print_step("Testing connection health")
            await ws.send(json.dumps({"id": 2, "type": "ping"}))
            msg = await ws.recv()
            pong = json.loads(msg)

            if pong.get("type") == "pong":
                self.print_success("Connection healthy (ping/pong working)")
            else:
                self.print_failure("Ping/pong failed")

    def morning_routine_scenario(self):
        """Test morning routine automation scenario"""
        self.print_scenario("Morning Routine Automation")

        self.print_step("Getting initial state of devices")
        initial_states = self.get_states()
        self.print_success(f"Found {len(initial_states)} devices")

        # Find lights and other devices
        lights = [s for s in initial_states if s["entity_id"].startswith("light.")]
        thermostats = [s for s in initial_states if s["entity_id"].startswith("climate.")]

        self.print_success(f"Found {len(lights)} lights, {len(thermostats)} thermostats")

        # Turn on bedroom light
        if lights:
            bedroom_light = next((l for l in lights if "bedroom" in l["entity_id"]), lights[0])
            entity_id = bedroom_light["entity_id"]

            self.print_step(f"Turning on {entity_id} for wake-up")
            self.call_service("light", "turn_on", entity_id, brightness=128)

            # Verify the change
            new_state = self.get_state(entity_id)
            if new_state["state"] == "on":
                self.print_success(f"{entity_id} is now on")
                brightness = new_state.get("attributes", {}).get("brightness", "unknown")
                print(f"    Brightness: {brightness}")
            else:
                self.print_failure(f"Failed to turn on {entity_id}")

        # Adjust thermostat for morning
        if thermostats:
            thermostat = thermostats[0]
            entity_id = thermostat["entity_id"]

            self.print_step(f"Setting {entity_id} to comfortable morning temperature")
            self.set_state(entity_id, "heat", {"temperature": 72})

            new_state = self.get_state(entity_id)
            temp = new_state.get("attributes", {}).get("temperature", "unknown")
            self.print_success(f"Thermostat set to {temp}°F")

    def security_check_scenario(self):
        """Test security system check scenario"""
        self.print_scenario("Evening Security Check")

        self.print_step("Checking all door and window sensors")
        states = self.get_states()

        sensors = [s for s in states if s["entity_id"].startswith("binary_sensor.")]
        locks = [s for s in states if s["entity_id"].startswith("lock.")]

        self.print_success(f"Found {len(sensors)} sensors, {len(locks)} locks")

        # Check sensor states
        open_sensors = [s for s in sensors if s["state"] == "on"]
        if open_sensors:
            self.print_failure(f"{len(open_sensors)} sensors are open:")
            for sensor in open_sensors:
                print(f"    • {sensor['entity_id']}")
        else:
            self.print_success("All sensors secure (closed)")

        # Check locks
        unlocked = [l for l in locks if l["state"] == "unlocked"]
        if unlocked:
            self.print_failure(f"{len(unlocked)} locks are unlocked:")
            for lock in unlocked:
                print(f"    • {lock['entity_id']}")

                # Lock them
                self.print_step(f"Locking {lock['entity_id']}")
                self.call_service("lock", "lock", lock["entity_id"])

                new_state = self.get_state(lock["entity_id"])
                if new_state["state"] == "locked":
                    self.print_success(f"{lock['entity_id']} is now locked")
        else:
            self.print_success("All locks are secured")

    def energy_saving_scenario(self):
        """Test energy saving automation scenario"""
        self.print_scenario("Energy Saving Mode - Away from Home")

        self.print_step("Getting all controllable devices")
        states = self.get_states()

        lights = [s for s in states if s["entity_id"].startswith("light.")]
        switches = [s for s in states if s["entity_id"].startswith("switch.")]
        thermostats = [s for s in states if s["entity_id"].startswith("climate.")]

        self.print_success(f"Found {len(lights)} lights, {len(switches)} switches, {len(thermostats)} thermostats")

        # Turn off all lights
        self.print_step("Turning off all lights")
        on_lights = [l for l in lights if l["state"] == "on"]
        for light in on_lights:
            self.call_service("light", "turn_off", light["entity_id"])
        self.print_success(f"Turned off {len(on_lights)} lights")

        # Turn off non-essential switches
        self.print_step("Turning off non-essential switches")
        on_switches = [s for s in switches if s["state"] == "on"]
        for switch in on_switches:
            self.call_service("switch", "turn_off", switch["entity_id"])
        self.print_success(f"Turned off {len(on_switches)} switches")

        # Set thermostats to eco mode
        self.print_step("Setting thermostats to energy-saving mode")
        for thermostat in thermostats:
            self.set_state(thermostat["entity_id"], "eco", {"temperature": 65})
        self.print_success(f"Set {len(thermostats)} thermostats to eco mode (65°F)")

    def entertainment_scenario(self):
        """Test movie night automation scenario"""
        self.print_scenario("Movie Night Setup")

        states = self.get_states()

        # Find relevant devices
        lights = [s for s in states if s["entity_id"].startswith("light.") and "living" in s["entity_id"]]
        media_players = [s for s in states if s["entity_id"].startswith("media_player.")]

        self.print_success(f"Found {len(lights)} living room lights, {len(media_players)} media players")

        # Dim the lights
        self.print_step("Dimming lights for movie watching")
        for light in lights:
            self.call_service("light", "turn_on", light["entity_id"], brightness=51)  # 20% brightness
            self.print_success(f"Dimmed {light['entity_id']} to 20%")

        # Start media player
        if media_players:
            player = media_players[0]
            self.print_step(f"Starting {player['entity_id']}")
            self.call_service("media_player", "turn_on", player["entity_id"])
            self.print_success(f"Media player ready")


def main():
    """Run all scenario tests"""
    print("\n" + "█" * 70)
    print("HOME ASSISTANT SCENARIO TESTING")
    print("Testing real-world automation workflows")
    print("█" * 70)

    tester = HomeAssistantScenarioTester()

    try:
        # Scenario 1: Morning routine
        tester.morning_routine_scenario()

        # Scenario 2: Security check
        tester.security_check_scenario()

        # Scenario 3: Energy saving
        tester.energy_saving_scenario()

        # Scenario 4: Entertainment
        tester.entertainment_scenario()

        # Scenario 5: WebSocket real-time monitoring
        asyncio.run(tester.websocket_scenario())

        # Final summary
        print("\n" + "█" * 70)
        print("✓ ALL SCENARIOS COMPLETED")
        print("█" * 70 + "\n")

    except requests.exceptions.ConnectionError:
        print("\n✗ Connection Error!")
        print("Make sure ISH server is running:")
        print("  ./ish seed homeassistant")
        print("  ./ish serve")
    except Exception as e:
        print(f"\n✗ Error: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    main()
