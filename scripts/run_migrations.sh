#!/bin/bash
set -e
DATABASE_URL="${DATABASE_URL:-postgres://user:password@localhost:5432/docstore?sslmode=disable}"
goose -dir migrations postgres "$DATABASE_URL" up
echo "Migrations done."