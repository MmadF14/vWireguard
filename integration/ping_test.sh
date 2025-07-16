#!/bin/bash
# Simple integration test for tunnel routing
set -e
ID=$1
wg-quick up client.conf
curl -m 5 -s -o /dev/null https://8.8.8.8 && echo "OK" || echo "FAIL"
wg-quick down client.conf
