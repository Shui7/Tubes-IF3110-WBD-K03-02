# Database Management Platform

A web application for managing PostgreSQL database connections, users, groups, and group access permissions. Users can sign in to browse available databases, while administrators can manage connections and access control.

## Project description

For Tugas Besar IF3110 Web Application Development

By K03-02:

- 13524003 Faiq Azzam Nafidz
- 13524018 Muhammad Nafis Habibi
- 13524072 I Gusti Ngurah Alit Dharma Y

## Run with Docker

### Prerequisites

- Docker Engine or Docker Desktop
- Docker Compose v2 (`docker compose`)

### Start the application

From the repository root, run:

```sh
docker compose up --build -d
```

Compose starts the web application, the platform PostgreSQL database, and a sample PostgreSQL database. Wait for the services to become healthy, then open:

<http://localhost:8080>

The application listens on port `8080` by default. The sample PostgreSQL server is published on port `5433` by default.

### Sign in

On a fresh platform database, the application seeds this development administrator account:

- Username: `admin`
- Password: `admin123`

Use this account for local development only. Change the default credentials before exposing the application to other users or networks.

### Useful Docker commands

View service status:

```sh
docker compose ps
```

Follow application logs:

```sh
docker compose logs -f app
```

Stop the services:

```sh
docker compose down
```

The PostgreSQL data is stored in named Docker volumes and is kept when you run `docker compose down`. To remove the database volumes and their data as well, run:

```sh
docker compose down --volumes
```

This permanently deletes the platform and sample database data.

## Configuration

Docker Compose reads environment variables from the shell or a `.env` file in the repository root. The platform database connection must use the Compose service name `dbplatform` as its host; containers should not use `localhost` or a host-mapped port to reach one another.

Common variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `POSTGRES_USER` | `postgres` | Platform database user |
| `POSTGRES_PASSWORD` | `postgres` | Platform database password |
| `POSTGRES_DB` | `dbplatform` | Platform database name |
| `DATABASE_URL` | Compose-configured connection to `dbplatform:5432` | Connection string used by the application |
| `APP_PORT` | `8080` | Host port for the web application |
| `DBSAMPLES_PORT` | `5433` | Host port for the sample PostgreSQL server |

If you override the platform database credentials, also set `DATABASE_URL` to matching values. For example:

```dotenv
POSTGRES_USER=postgres
POSTGRES_PASSWORD=replace-with-a-local-password
POSTGRES_DB=dbplatform
DATABASE_URL=host=dbplatform port=5432 dbname=dbplatform user=postgres password=replace-with-a-local-password sslmode=disable
```

## Development

Run the Go test suite:

```sh
go test ./...
```

The platform schema is in [`db/platform/schema.sql`](db/platform/schema.sql). Sample database SQL files are loaded by [`db/init-samples.sh`](db/init-samples.sh) when the sample database volume is initialized for the first time.
