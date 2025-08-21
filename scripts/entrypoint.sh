#!/bin/sh
set -e

# Environment variables (expected to be passed from Docker Compose or Kubernetes)
DB_URL=${DB_URL:-"postgresql://user:password@db:5432/mydatabase?sslmode=disable"}

# Extract DB connection parameters from DB_URL
DB_PROTOCOL=$(echo "$DB_URL" | awk -F:// '{print $1}')
DB_USER_PASS_HOST_PORT_DB=$(echo "$DB_URL" | awk -F:// '{print $2}')
DB_USER_PASS=$(echo "$DB_USER_PASS_HOST_PORT_DB" | awk -F@ '{print $1}')
DB_HOST_PORT_DB=$(echo "$DB_USER_PASS_HOST_PORT_DB" | awk -F@ '{print $2}')

DB_USER=$(echo "$DB_USER_PASS" | awk -F: '{print $1}')
DB_PASSWORD=$(echo "$DB_USER_PASS" | awk -F: '{print $2}')

DB_HOST=$(echo "$DB_HOST_PORT_DB" | awk -F: '{print $1}')
DB_PORT_DB=$(echo "$DB_HOST_PORT_DB" | awk -F: '{print $2}')
DB_PORT=$(echo "$DB_PORT_DB" | awk -F/ '{print $1}')
DB_NAME_QUERY=$(echo "$DB_PORT_DB" | awk -F/ '{print $2}')
DB_NAME=$(echo "$DB_NAME_QUERY" | awk -F? '{print $1}')


# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to start at $DB_HOST:$DB_PORT..."
export PGPASSWORD="$DB_PASSWORD"
timeout=60
while ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' > /dev/null 2>&1; do
  timeout=$(($timeout - 1))
  if [ $timeout -eq 0 ]; then
    echo "PostgreSQL connection timeout!"
    exit 1
  fi
  sleep 1
done
unset PGPASSWORD
echo "PostgreSQL started."

echo "Running database migrations..."
migrate -path /app/db/migration -database "$DB_URL" up
echo "Database migrations completed."

echo "Starting application..."
exec "$@" # Execute the command passed to the entrypoint (e.g., /app/main)
