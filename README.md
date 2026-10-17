# traefik-webui

A web UI to monitor Traefik and edit its configuration. Go backend, Svelte frontend embedded in a single binary. See [REQUIREMENTS.md](REQUIREMENTS.md) for scope and decisions.

Status: skeleton. It serves the frontend, proxies read-only Traefik API calls at `/api/traefik/*`, and exposes `/healthz`.

## Develop

```sh
cd web && npm install && npm run build   # builds into internal/server/dist
cd .. && TRAEFIK_API_URL=http://localhost:8080 go run ./cmd/traefik-webui
```

Settings: `TRAEFIK_API_URL` (default `http://traefik:8080`), `LISTEN_ADDR` (default `:8081`).

## License

MIT
