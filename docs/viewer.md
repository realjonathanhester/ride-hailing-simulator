# Viewer (`rhs view`)

See the [tooling specification](tooling_spec.md) for every page and endpoint.

`rhs view` serves a local results viewer (dashboard, trips, supply and run comparison) on `127.0.0.1`. See the [tooling specification](tooling_spec.md).

```sh
go run ./cmd/rhs run -settings tests/fixtures/settings/base_default.toml \
  -supply tests/fixtures/supply/one_car_at_kezar_stadium.csv -requests tests/fixtures/demand/simple_pool.csv -output out/a
go run ./cmd/rhs run -settings tests/fixtures/settings/pooling_enabled.toml \
  -supply tests/fixtures/supply/one_car_at_kezar_stadium.csv -requests tests/fixtures/demand/simple_pool.csv -output out/b
GOOGLE_MAPS_API_KEY=... go run ./cmd/rhs view -output out/a -compare out/b
# open http://127.0.0.1:8080/
```

Maps need a Google Maps JavaScript API key from `GOOGLE_MAPS_API_KEY` or a local, git-ignored `viewer.toml`:

```toml
[viewer]
google_maps_api_key = "..."
google_maps_map_id = "..."  # optional: vector maps and advanced markers
display_timezone = "America/Chicago"  # optional: zone for local times (default: the run's timezone, then the browser's)
```

Without a key every page still works and maps show a notice. Flags: `-output` (required), `-compare`, `-port` (default 8080), `-settings` (default `viewer.toml`), `-timezone` (IANA zone for local times).

