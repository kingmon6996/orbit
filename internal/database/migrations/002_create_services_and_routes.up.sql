CREATE TABLE services (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX services_enabled_idx ON services (enabled);

CREATE TABLE routes (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    method TEXT NOT NULL,
    service_id UUID NOT NULL REFERENCES services (id) ON DELETE RESTRICT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    strip_prefix BOOLEAN NOT NULL DEFAULT FALSE,
    rewrite_path TEXT NOT NULL DEFAULT '',
    timeout BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (method, path)
);

CREATE INDEX routes_lookup_idx ON routes (method, path);
CREATE INDEX routes_enabled_idx ON routes (enabled);
CREATE INDEX routes_service_id_idx ON routes (service_id);