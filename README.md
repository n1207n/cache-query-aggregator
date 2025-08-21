# golang-app-scaffold

This project is a scaffold for a full-stack application using Go (with Gin framework) for the backend, PostgreSQL as the database, and Redis for caching/session management.

It includes Docker setup for containerization and Docker Compose for orchestrating services.

## Features Implemented
*   **Backend API**: Built with Go and Gin.
*   **Database**: PostgreSQL with `pgx` driver.
*   **ORM/Query Builder**: `sqlc` for generating type-safe Go code from SQL queries.
*   **Database Migrations**: `golang-migrate/migrate` for managing database schema changes.
*   **Caching/Session**: Redis client setup.
*   **Configuration**: Environment variable-based configuration loading.
*   **User Management**: Basic User model, repository, service, and handlers (CreateUser, GetUserByID).
*   **Containerization**:
    *   Multi-stage `Dockerfile` for development (with Air for live reload, Delve for debugging) and production.
    *   `compose.yml` to run the application, PostgreSQL, and Redis together.
    *   Entrypoint script (`scripts/entrypoint.sh`) for the production container to check DB connection and run migrations.
*   **Live Reloading**: `.air.toml` configured for development.

## Project Structure (Simplified)
```text
. 
├── cmd/server/main.go    # Application entry point
├── config/               # Configuration loading
├── db/
│   ├── migration/        # Database migration files (.sql)
│   ├── queries/          # SQL queries for sqlc
│   └── sqlc/             # Generated Go code by sqlc
├── internal/
│   ├── handler/          # HTTP handlers (Gin)
│   ├── repository/       # Database interaction logic
│   ├── router/           # API route definitions
│   └── service/          # Business logic
├── scripts/
│   └── entrypoint.sh     # Docker entrypoint script for prod
├── .air.toml             # Air configuration for live reload
├── Dockerfile            # Docker build instructions
├── compose.yml           # Docker Compose configuration
├── go.mod                # Go module definition
├── sqlc.yaml             # sqlc configuration
└── README.md             # This file
```

## Prerequisites
*   Docker Desktop (or Docker Engine + Docker Compose)
*   Go (version 1.24+ for local development if not using Docker exclusively)
*   Make (optional, for Makefile commands if added in the future)

## Getting Started

### 1. Clone the Repository

```bash
git clone <your-repository-url>
cd <your-project-directory>
```

### 2. Environment Setup

Copy the example environment file and customize it if necessary:
```bash
cp .env.example .env
```

### 3. Running the Application

#### Production-like Environment
This command builds the production-ready image and starts all services (app, db, redis). The entrypoint script will automatically run database migrations.

```bash
docker compose up --build
```
The application will be accessible at http://localhost:${APP_PORT} (default: http://localhost:8080).

#### Development with Live Reload (Air)
For development, you can use the dev override compose file to enable live reloading with Air.

```bash
docker compose -f compose.yml -f compose.dev.yml up --build
```

Changes to Go files (and other specified extensions in .air.toml) will trigger an automatic rebuild and restart of the application.

#### Development with Debugging (Delve)

Similarly, for debugging with Delve, you can use the debug override compose file to enable delve debugger with breakpoints.

```bash
docker compose -f compose.yml -f compose.debug.yml up --build
```

You can then attach your Go IDE's debugger to localhost:2345.

## Database Migrations (golang-migrate/migrate)

Migration files are located in `db/migration/`. The migrate CLI tool is included in the Docker images (dev-common and prod stages).

### Creating a New Migration

To create a new migration file (e.g., add_new_feature), run the following command locally if you have migrate installed, or execute it inside a running app container:

#### Locally (if migrate CLI is installed)
```bash
migrate create -ext sql -dir db/migration -seq add_new_feature
```

#### Inside the app container (ensure the app service is running, e.g., via dev compose setup)
```bash
docker compose exec app migrate create -ext sql -dir /app/db/migration -seq add_new_feature
```

This will create two files in `db/migration/` (e.g., 00000X_add_new_feature.up.sql and 00000X_add_new_feature.down.sql). Edit these files to define your schema changes.

#### Running Migrations Manually

The entrypoint.sh script in the prod target automatically runs migrate ... up on container startup. However, you can run migrations manually against the database service managed by Docker Compose.

Ensure your .env file is configured, especially DB_URL (or POSTGRES_URL from which DB_URL is derived for the entrypoint). The migrate command inside the container will use the DB_URL environment variable.
- Apply all pending up migrations:
   ```bash
   docker compose exec app migrate -path /app/db/migration -database "$DB_URL" up
   ```
- Rollback the last down migration:
   ```bash
   docker compose exec app migrate -path /app/db/migration -database "$DB_URL" down 1
   ```
- Migrate to a specific version:
   ```bash
   docker compose exec app migrate -path /app/db/migration -database "$DB_URL" goto VERSION_NUMBER
   ```
- Force a specific version (useful for fixing dirty state):
   ```bash
   docker compose exec app migrate -path /app/db/migration -database "$DB_URL" force VERSION_NUMBER
   ```

## SQLC (SQL Code Generation)

sqlc generates Go code from your SQL queries located in `db/queries/`. The sqlc CLI is included in the Docker images. Configuration is in sqlc.yaml.

### Generating Go Code
After making changes to your SQL queries in `db/queries/` or updating `sqlc.yaml`, you need to regenerate the Go code.
```bash
docker compose exec app sqlc generate --path /app/sqlc.yaml
```

The generated Go files will be placed in `db/sqlc/` as specified in `sqlc.yaml`. Remember to commit these generated files to your repository.

## API Endpoints
Currently implemented user endpoints (base path /api/v1):
- POST /users: Create a new user.
- GET /users/:id: Get a user by their ID.

(More to be added)

Future Work / Enhancements
- Implement full CRUD operations for users.
- Add authentication (e.g., JWT).
- Implement authorization/roles.
- Add more services and features.
- Write unit and integration tests.
- Set up CI/CD pipeline.
- Improve error handling and logging.