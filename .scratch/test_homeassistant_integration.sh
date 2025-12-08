#!/bin/bash
# Integration test for Home Assistant plugin

set -e

echo "=== Testing Home Assistant Plugin ==="

# Setup
rm -f test_homeassistant.db
./ish serve -d test_homeassistant.db -p 18823 &
SERVER_PID=$!
sleep 2

# Test 1: Set an entity state
echo ""
echo "1. Setting light.living_room state to 'on'..."
curl -s -X POST -H "Authorization: Bearer token_home_main" \
  -H "Content-Type: application/json" \
  -d '{"state":"on","attributes":{"brightness":255,"color_temp":370}}' \
  http://localhost:18823/api/states/light.living_room

# Test 2: Get the entity state back
echo ""
echo "2. Getting light.living_room state..."
curl -s -H "Authorization: Bearer token_home_main" \
  http://localhost:18823/api/states/light.living_room

# Test 3: Call a service
echo ""
echo "3. Calling light.turn_on service..."
curl -s -X POST -H "Authorization: Bearer token_home_main" \
  -H "Content-Type: application/json" \
  -d '{"entity_id":"light.living_room","service_data":{"brightness":255}}' \
  http://localhost:18823/api/services/light/turn_on

# Test 4: Check admin UI shows the instance
echo ""
echo "4. Checking admin UI for instances..."
response=$(curl -s http://localhost:18823/admin/plugins/homeassistant/instances)
echo "$response" | grep -q "default" && echo "✓ Instance 'default' found" || (echo "✗ Instance NOT found"; kill $SERVER_PID; exit 1)

# Test 5: Check admin UI shows the entity
echo ""
echo "5. Checking admin UI for entities..."
response=$(curl -s http://localhost:18823/admin/plugins/homeassistant/entities)
echo "$response" | grep -q "light.living_room" && echo "✓ Entity 'light.living_room' found" || (echo "✗ Entity NOT found"; kill $SERVER_PID; exit 1)

# Test 6: Check admin UI shows the state
echo ""
echo "6. Checking admin UI for states..."
response=$(curl -s http://localhost:18823/admin/plugins/homeassistant/states)
echo "$response" | grep -q "light.living_room" && echo "✓ State found" || (echo "✗ State NOT found"; kill $SERVER_PID; exit 1)

# Test 7: Check admin UI shows the service call
echo ""
echo "7. Checking admin UI for service calls..."
response=$(curl -s http://localhost:18823/admin/plugins/homeassistant/service_calls)
echo "$response" | grep -q "light" && echo "✓ Service call found" || (echo "✗ Service call NOT found"; kill $SERVER_PID; exit 1)

# Cleanup
kill $SERVER_PID 2>/dev/null
rm -f test_homeassistant.db

echo ""
echo "=== All Home Assistant Tests Passed! ==="
