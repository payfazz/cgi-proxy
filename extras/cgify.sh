#!/bin/sh

cd "$(dirname "$0")"

echo "Content-Type: application/octet-stream"
echo ""

exec "$@"
