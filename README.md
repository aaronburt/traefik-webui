# traefik-webui

A web UI to monitor Traefik and edit its configuration. Go backend, Svelte frontend embedded in a single binary. See [REQUIREMENTS.md](REQUIREMENTS.md) for scope and decisions.

Status: early. It serves the frontend, has single-user password login, proxies read-only Traefik API calls at `/api/traefik/*` (login required), and exposes `/healthz`.

## Login

On first run the UI asks you to create the admin password (12 to 72 characters). Only a bcrypt hash is stored, in `DATA_DIR/auth.json` (mode 0600). Open the UI right after starting it for the first time, since whoever reaches the setup page first sets the password. Sessions are HttpOnly, SameSite=Strict cookies that last 12 hours, and repeated failed logins from one address are locked out for a minute.

## Develop

```sh
cd web && npm install && npm run build   # builds into internal/server/dist
cd .. && TRAEFIK_API_URL=http://localhost:8080 go run ./cmd/traefik-webui
```

Settings: `TRAEFIK_API_URL` (default `http://traefik:8080`), `LISTEN_ADDR` (default `:8081`), `DATA_DIR` (default `./data`, `/data` in the image; mount a volume there).

## License

MIT
