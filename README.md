# Orbit

Orbit is a high-performance enterprise API Gateway written in Go.

## Current Status

Module 1 - Foundation

## Requirements

Go. Docker, Kubernetes, and external infrastructure are not required.

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

Environment variables are the source of truth. `.env.example` documents the supported values; a real `.env` is not required.

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

Authentication, authorization, rate limiting, load balancing, and reverse proxying are planned for later modules.