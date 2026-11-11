#!/bin/sh
set -eu

# Den Schlüssel für die Verschlüsselung der Datenbank aus einem Docker Secret lesen.
if [ -f /run/secrets/todo_encryption_key ]; then
  ENCRYPTION_KEY="$(cat /run/secrets/todo_encryption_key)"
  export ENCRYPTION_KEY
fi

exec /app/todo -addr "${LISTEN_ADDR:-:8080}" -db "${DB_PATH:-/data/todo.db}"
