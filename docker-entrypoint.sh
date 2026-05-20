#!/bin/sh
set -eu

exec /app/todo -addr "${LISTEN_ADDR:-:8080}" -db "${DB_PATH:-/data/todo.db}"
