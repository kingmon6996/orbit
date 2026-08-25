# Orbit

Orbit is a high-performance enterprise API Gateway written in Go.

## Current Status

Module 2 - PostgreSQL & Persistence Foundation

## Requirements

Go and PostgreSQL when persistence is enabled. Docker and Kubernetes are not required.

## Run

```sh
go run ./cmd/orbit
```

## Build

```sh
go build -o orbit ./cmd/orbit
```

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `APP_NAME` | `orbit` | Application name |
| `APP_ENV` | `development` | `development`, `staging`, or `production` |
| `ORBIT_HOST` | `0.0.0.0` | Listen host |
| `ORBIT_PORT` | `8080` | Listen port |
| `ORBIT_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `ORBIT_READ_TIMEOUT` | `15s` | Request read timeout |
| `ORBIT_WRITE_TIMEOUT` | `15s` | Response write timeout |
| `ORBIT_IDLE_TIMEOUT` | `60s` | Keep-alive timeout |
| `ORBIT_READ_HEADER_TIMEOUT` | `5s` | Header read timeout |
| `ORBIT_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |
| `DATABASE_ENABLED` | `false` | Require PostgreSQL at startup |
| `DATABASE_URL` | empty | PostgreSQL connection URL |
| `DATABASE_MAX_CONNS` | `20` | Maximum pool connections |
| `DATABASE_MIN_CONNS` | `2` | Minimum pool connections |
| `DATABASE_MAX_CONN_LIFETIME` | `30m` | Maximum connection lifetime |
| `DATABASE_MAX_CONN_IDLE_TIME` | `10m` | Maximum idle connection time |
| `DATABASE_HEALTH_CHECK_PERIOD` | `30s` | Pool health-check period |

Environment variables are the source of truth. `.env.example` documents the supported values; a real `.env` is not required.

### PostgreSQL

Set `DATABASE_ENABLED=true` and provide `DATABASE_URL` to require PostgreSQL during Orbit startup. The URL should contain the database credentials supplied by the deployment environment and is never logged. Pool settings are controlled by `DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS`, `DATABASE_MAX_CONN_LIFETIME`, `DATABASE_MAX_CONN_IDLE_TIME`, and `DATABASE_HEALTH_CHECK_PERIOD`.

Apply or roll back the embedded, tracked migrations with:

```sh
DATABASE_ENABLED=true go run ./cmd/migrate up
DATABASE_ENABLED=true go run ./cmd/migrate down
```

On PowerShell, set the variables with `$env:DATABASE_ENABLED="true"` and `$env:DATABASE_URL="..."` before running the command. The default `DATABASE_ENABLED=false` keeps local liveness development independent of PostgreSQL.

## Endpoints

- `GET /` returns application metadata.
- `GET /health` returns process liveness.

## Development

```sh
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
```

Repository integration tests should be run with a real PostgreSQL instance and `DATABASE_URL` configured. Authentication, authorization, rate limiting, load balancing, and reverse proxying are planned for later modules.

## Routing

Module 3 loads enabled services and routes from PostgreSQL once at startup into an immutable in-memory snapshot. Requests do not query PostgreSQL. A future controlled reload can replace the snapshot atomically.

Routes contain a name, HTTP method, path, service ID, enabled state, prefix/rewrite fields, and timeout. Supported paths are exact paths such as `/api/users`, parameter paths such as `/api/users/{id}`, and final-segment wildcards such as `/api/users/*`. Trailing slashes are normalized away except for `/`; query strings do not participate in matching.

Methods are normalized to uppercase and supported methods are `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, and `HEAD`. Precedence is exact, then parameter, then wildcard. Duplicate method/path patterns are rejected, and a path with an unsupported method returns `405` with `Allow`; unknown paths return `404`.

For example, a service named `user-api` can own a route with method `GET`, path `/api/users/{id}`, and its service UUID as `service_id`. Module 3 returns a temporary JSON match result; backend forwarding is not implemented yet.