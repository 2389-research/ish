#!/usr/bin/env python3
# ABOUTME: Test script for Home Assistant WebSocket API
# ABOUTME: Demonstrates authentication, get_states, and ping/pong functionality

import asyncio
import websockets
import json


async def test_homeassistant_websocket():
    """Test Home Assistant WebSocket API implementation"""
    uri = "ws://localhost:9000/api/websocket"

    print("=" * 60)
    print("Home Assistant WebSocket API Test")
    print("=" * 60)

    try:
        async with websockets.connect(uri) as ws:
            print("\n✓ WebSocket connection established")

            # Step 1: Receive auth_required
            print("\n1. Waiting for auth_required...")
            msg = await ws.recv()
            auth_req = json.loads(msg)
            print(f"   < {json.dumps(auth_req, indent=2)}")

            if auth_req.get("type") != "auth_required":
                print("   ✗ Expected auth_required, got:", auth_req.get("type"))
                return
            print("   ✓ Received auth_required")

            # Step 2: Send auth
            print("\n2. Sending authentication...")
            auth_msg = {
                "type": "auth",
                "access_token": "token_home_main"
            }
            await ws.send(json.dumps(auth_msg))
            print(f"   > {json.dumps(auth_msg, indent=2)}")

            # Step 3: Receive auth_ok
            print("\n3. Waiting for auth_ok...")
            msg = await ws.recv()
            auth_ok = json.loads(msg)
            print(f"   < {json.dumps(auth_ok, indent=2)}")

            if auth_ok.get("type") != "auth_ok":
                print("   ✗ Expected auth_ok, got:", auth_ok.get("type"))
                return
            print("   ✓ Authentication successful!")

            # Step 4: Get states
            print("\n4. Requesting all entity states...")
            get_states_msg = {
                "id": 1,
                "type": "get_states"
            }
            await ws.send(json.dumps(get_states_msg))
            print(f"   > {json.dumps(get_states_msg, indent=2)}")

            # Step 5: Receive states
            print("\n5. Receiving states response...")
            msg = await ws.recv()
            states_response = json.loads(msg)

            if states_response.get("success"):
                states = states_response.get("result", [])
                print(f"   ✓ Received {len(states)} entity states")

                if states:
                    print("\n   Sample entities:")
                    for state in states[:5]:
                        entity_id = state.get("entity_id", "unknown")
                        current_state = state.get("state", "unknown")
                        print(f"     - {entity_id}: {current_state}")
                else:
                    print("   ⚠ No entities found (database may be empty)")
                    print("   Hint: Run './ish seed' to populate test data")
            else:
                print(f"   ✗ get_states failed: {states_response}")

            # Step 6: Ping/Pong test
            print("\n6. Testing ping/pong...")
            ping_msg = {
                "id": 2,
                "type": "ping"
            }
            await ws.send(json.dumps(ping_msg))
            print(f"   > {json.dumps(ping_msg, indent=2)}")

            msg = await ws.recv()
            pong_response = json.loads(msg)
            print(f"   < {json.dumps(pong_response, indent=2)}")

            if pong_response.get("type") == "pong":
                print("   ✓ Ping/pong working correctly")
            else:
                print(f"   ✗ Expected pong, got: {pong_response.get('type')}")

            print("\n" + "=" * 60)
            print("✓ All tests passed!")
            print("=" * 60)

    except websockets.exceptions.ConnectionRefusedError:
        print("\n✗ Connection refused!")
        print("Make sure ISH server is running:")
        print("  ./ish serve")
    except Exception as e:
        print(f"\n✗ Error: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    asyncio.run(test_homeassistant_websocket())
