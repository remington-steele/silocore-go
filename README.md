# Silocore Go

A multi-tenant SAAS application using Go, Chi, go-migrate, templ templates, HTMX, Tailwind CSS, Docker, and Postgres.

## Features

- Multi-tenant architecture with Row Level Security (RLS) for data isolation
- JWT-based authentication and authorization
- Role-based access control at both platform and tenant levels
- Server-side rendering with templ templates
- Dynamic UI interactions with HTMX
- Modern styling with Tailwind CSS

## Database Migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations. The migration files are located in the `sql/migrations` directory.

### Prerequisites

- Go 1.21 or later
- PostgreSQL 15 or later
- Make (optional, for using the Makefile commands)

### Environment Variables

Create a `.env` file in the root directory with the following variables:

```
# Regular application database connection (limited privileges)
DATABASE_URL=postgres://silocore_app_user:silocore@localhost:5432/silocore

# Admin database connection for migrations (schema modification privileges)
DATABASE_ADMIN_URL=postgres://silocore_admin_user:silocore@localhost:5432/silocore

# JWT secret for authentication
JWT_SECRET=your_jwt_secret_key_change_this_in_production
JWT_EXPIRATION_SECONDS=86400
JWT_REFRESH_EXPIRATION_SECONDS=604800
JWT_ISSUER=silocore
```

### Running Migrations

You can run migrations using the following commands:

#### Using Make

```bash
# Run all migrations up
make migrate

# Run all migrations down
make migrate-down

# Run a specific number of migrations up
make migrate-steps steps=1

# Run a specific number of migrations down
make migrate-down-steps steps=1
```

#### Using the Migration Tool Directly

```bash
# Build the migration tool
go build -o bin/migrate cmd/migrate/main.go

# Run all migrations up
./bin/migrate

# Run all migrations down
./bin/migrate -down

# Run a specific number of migrations up
./bin/migrate -steps 1

# Run a specific number of migrations down
./bin/migrate -down -steps 1

# Specify a custom migrations path
./bin/migrate -path /path/to/migrations
```

### Migration Files

Migration files are located in the `sql/migrations` directory. Each migration file should be named in the format `{version}_{name}.sql`, where `{version}` is a numeric version and `{name}` is a descriptive name for the migration.

The current migration files are:

- `20240307001_set_sqlx_table_owner.sql`: Sets the owner of the _sqlx_migrations table.
- `20240307002_initial_schema.sql`: Creates the initial database schema.
- `20240307003_row_level_security.sql`: Implements row-level security for tenant isolation.

## Architecture

See [architecture.md](architecture.md) for details on the architecture of the application.

## Implementation Steps

See [steps.md](steps.md) for the implementation steps and progress tracking.

## Prerequisites

- Go (latest stable version)
- PostgreSQL 12+
- Docker (optional, for containerized deployment)

## JWT Configuration

The application uses JSON Web Tokens (JWT) for authentication and tenant context management. The following environment variables are used for JWT configuration:

- `JWT_SECRET`: Secret key used for signing and verifying JWT tokens. This should be a strong, random string.
- `JWT_EXPIRATION_SECONDS`: Access token expiration time in seconds. Defaults to 86400 (24 hours) if not specified.
- `JWT_REFRESH_EXPIRATION_SECONDS`: Refresh token expiration time in seconds. Defaults to 7 times the access token expiration (7 days) if not specified.
- `JWT_ISSUER`: Issuer claim value for the JWT tokens. Defaults to "silocore" if not specified.

### JWT Token Structure

The JWT tokens include the following claims:

- Standard JWT claims:
  - `iss`: Issuer of the token
  - `iat`: Issued at timestamp
  - `exp`: Expiration timestamp

- Custom claims:
  - `user_id`: Unique identifier for the authenticated user
  - `username`: Username of the authenticated user
  - `tenant_id`: Optional tenant ID for tenant context (omitted for global context)

The system uses two types of tokens:

1. **Access Token**: Short-lived token used for authentication and authorization. Includes tenant context if applicable.
2. **Refresh Token**: Longer-lived token used to obtain new access tokens. Does not include tenant context for security reasons.

### Tenant Context Switching

Admin users can switch tenant contexts by selecting a tenant from the UI, which updates the JWT token with the new tenant context. System-wide data access is facilitated by omitting the `tenant_id` in the JWT for admin routes.

## Getting Started

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/silogit
   cd silocore
   ```

2. Set up the PostgreSQL database:
   ```
   psql -U postgres -f sql/init/init.sql
   ```

3. Run the application:
   ```
   cargo run
   ```