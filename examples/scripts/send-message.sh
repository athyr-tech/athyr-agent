#!/bin/bash
#
# send-message.sh - Send a message to a running athyr-agent via HTTP API
#
# Usage:
#   ./send-message.sh <subject> <message> [session_id]
#   ./send-message.sh test.input "Hello, world!"
#   ./send-message.sh chat.input "What is my name?" user-123
#
# Environment variables:
#   ATHYR_SERVER - Athyr HTTP API (default: http://localhost:8080)
#

set -e

ATHYR_SERVER="${ATHYR_SERVER:-http://localhost:8080}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

usage() {
    echo "Usage: $0 <subject> <message> [session_id]"
    echo ""
    echo "Examples:"
    echo "  $0 test.input 'Hello, world!'"
    echo "  $0 chat.input 'My name is Alice' user-123"
    echo ""
    echo "Environment:"
    echo "  ATHYR_SERVER  Athyr HTTP API (default: http://localhost:8080)"
    exit 1
}

if [ $# -lt 2 ]; then
    usage
fi

SUBJECT="$1"
MESSAGE="$2"
SESSION_ID="${3:-}"

# Build the message payload
if [ -n "$SESSION_ID" ]; then
    PAYLOAD=$(printf '{"session_id":"%s","content":"%s"}' "$SESSION_ID" "$MESSAGE")
else
    PAYLOAD="$MESSAGE"
fi

# Base64 encode
DATA=$(printf '%s' "$PAYLOAD" | base64 | tr -d '\n')

echo -e "${BLUE}Sending message...${NC}"
echo -e "  Server:  ${ATHYR_SERVER}"
echo -e "  Subject: ${SUBJECT}"
echo -e "  Message: ${MESSAGE}"
[ -n "$SESSION_ID" ] && echo -e "  Session: ${SESSION_ID}"
echo ""

RESPONSE=$(curl -s -X POST "${ATHYR_SERVER}/v1/publish" \
    -H "Content-Type: application/json" \
    -d "{\"subject\":\"${SUBJECT}\",\"data\":\"${DATA}\"}")

if echo "$RESPONSE" | grep -q '"ok":true'; then
    echo -e "${GREEN}Message sent.${NC}"
else
    echo -e "${RED}Error:${NC} $RESPONSE"
    exit 1
fi
