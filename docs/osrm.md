# Real roads (OSRM)

The default router drives straight lines. For real roads, run an [OSRM](https://project-osrm.org/) server; the setup script does it with Docker (the only requirement) and needs a few minutes and about 2 GB of RAM for Austin:

```sh
scripts/osrm_setup.sh --region austin
```

It downloads the [Geofabrik Texas extract](https://download.geofabrik.de/north-america/us/texas-latest.osm.pbf), crops it to the Austin metro box `-98.20,29.90,-97.33,30.82` with osmium (a Docker image if osmium isn't installed), prepares it for MLD with the car profile, starts `osrm-routed` on `127.0.0.1:5000`, smoke-tests a route, and writes `.osrm-data/austin/metadata.json` (source URL, download date, SHA-256, bbox, OSRM version). Each step is skipped when its output exists. Other regions: `--region NAME --pbf-url URL --bbox minlng,minlat,maxlng,maxlat`. Set `DOCKER="sudo docker"` if your user can't reach Docker, or `DOCKER_NETWORK=host` where Docker has no bridge network. Then add what it prints to your settings:

```toml
[hooks]
routing = "osrm"

[osrm]
url = "http://127.0.0.1:5000"
dataset_metadata_path = ".osrm-data/austin/metadata.json"
cache_path = "osrm_cache.jsonl"   # optional: lets a rerun go without the server
```

`OSRM_URL=http://127.0.0.1:5000 go test ./routing/osrm/` also runs a check against the real server. Stop it with `docker rm -f rhs-osrm-austin`.

