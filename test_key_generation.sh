#!/bin/bash

echo "=== Testing WireGuard Key Generation ==="

# Test the test endpoint
echo "1. Testing key derivation endpoint..."
curl -s "http://localhost:8080/api/test-key-derivation" | jq .

echo -e "\n2. Testing manual key generation..."
curl -s -X POST -H "Content-Type: application/json" \
     -d '{}' \
     "http://localhost:8080/api/generate-keypair" | jq .

echo -e "\n3. Testing PreShared key generation..."
curl -s -X POST -H "Content-Type: application/json" \
     "http://localhost:8080/api/generate-preshared-key" | jq .

echo -e "\nTest complete!" 