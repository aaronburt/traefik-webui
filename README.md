# traefik-webui

A web UI to monitor Traefik and edit its configuration. Go backend, Svelte frontend embedded in a single binary. See [REQUIREMENTS.md](REQUIREMENTS.md) for scope and decisions.

Status: single-user password authentication with rotation, dynamic file-provider configuration management (validation, diff preview, atomic writes, automated backups and rollback), CSV audit logging, read-only Traefik object inspection, and defensive HTTP security headers.

## Login

On first run the UI asks you to create the admin password (12 to 72 characters). Only a bcrypt hash is stored, in `DATA_DIR/auth.json` (mode 0600). The password can be rotated at any time under the Settings page. Open the UI right after starting it for the first time, since whoever reaches the setup page first sets the password. Sessions are HttpOnly, SameSite=Strict cookies that last 12 hours, and repeated failed logins from one address are locked out for a minute.

## Audit Log

Administrative and write actions are recorded to `DATA_DIR/audit.csv` and can be viewed or downloaded from the Audit Log tab in the UI.

## Develop

```sh
cd web && npm install && npm run build   # builds into internal/server/dist
cd .. && TRAEFIK_API_URL=http://localhost:8080 go run ./cmd/traefik-webui
```

Settings: `TRAEFIK_API_URL` (default `http://traefik:8080`), `LISTEN_ADDR` (default `:8081`), `DATA_DIR` (default `./data`, `/data` in the image; mount a volume there).

## License

MIT
