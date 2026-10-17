# traefik-webui

A web UI to monitor Traefik and edit its configuration. Go backend, Svelte frontend embedded in a single binary. See [REQUIREMENTS.md](REQUIREMENTS.md) for scope and decisions.

Status: single-user password authentication with rotation, dynamic file-provider configuration management (validation, diff preview, atomic writes, automated backups and rollback), CSV audit logging, read-only Traefik object inspection, and defensive HTTP security headers.

## Login

On first run the UI asks you to create the admin password (12 to 72 characters). Only a bcrypt hash is stored, in `DATA_DIR/auth.json` (mode 0600). The password can be rotated at any time under the Settings page. Open the UI right after starting it for the first time, since whoever reaches the setup page first sets the password. Sessions are HttpOnly, SameSite=Strict cookies that last 12 hours, and repeated failed logins from one address are locked out for a minute.

## Audit Log

Administrative and write actions are recorded to `DATA_DIR/audit.csv` and can be viewed or downloaded from the Audit Log tab in the UI.

## Configure

The UI has no built-in knowledge of any Traefik setup. Point it at your instance with environment variables:

| Variable | Required | Default | Meaning |
| --- | --- | --- | --- |
| `TRAEFIK_API_URL` | yes | none | Base URL of Traefik's API, for example `http://traefik:8080`. The API must be enabled in Traefik. |
| `CONFIG_DIR` | yes | none | The directory Traefik's file provider watches, as seen from inside the UI. It must already exist and be the same directory Traefik reads. |
| `DATA_DIR` | no | `./data` | Where the admin password, audit log and config backups are kept. The image sets `/data`; mount a volume there. |
| `LISTEN_ADDR` | no | `:8081` | Address the UI listens on. |

The server exits at startup and names any missing or invalid setting.

## Run next to Traefik

Run the UI as its own container, with the same config directory mounted into both containers:

```yaml
services:
  traefik:
    image: traefik:v3
    command:
      - --api
      - --providers.file.directory=/etc/traefik/dynamic
    volumes:
      - ./dynamic:/etc/traefik/dynamic

  traefik-webui:
    image: traefik-webui   # build from this repository
    environment:
      TRAEFIK_API_URL: http://traefik:8080
      CONFIG_DIR: /etc/traefik/dynamic
    volumes:
      - ./dynamic:/etc/traefik/dynamic
      - webui-data:/data
    ports:
      - "8081:8081"

volumes:
  webui-data:
```

## Develop

```sh
cd web && npm install && npm run build   # builds into internal/server/dist
cd .. && TRAEFIK_API_URL=http://localhost:8080 CONFIG_DIR=./dynamic go run ./cmd/traefik-webui
```

For the Vite dev server, set `TRAEFIK_WEBUI_URL` to the running UI (default `http://localhost:8081`).

## Tests

GitHub Actions (`.github/workflows/ci.yml`) runs on every pull request and every push to `main`:

- Go: `gofmt`, `go vet` and `go test -race ./...`.
- Web: `npm ci`, `svelte-check` and the production build.
- Browser smoke test (`e2e/`): Playwright drives the built UI against a small mock of Traefik's API, covering first-run setup, the objects tab, editing a config file with review and apply, Quick route validation and sign-in.
- Docker: builds the image, checks that it refuses to start without its required settings, then starts it and calls `/healthz`.

To run them locally: `go test ./...` and `cd web && npm run check`. For the browser test, build the UI and the binary (`npm run build` in `web/`, then `go build -o bin/traefik-webui ./cmd/traefik-webui`), then run `npx playwright install chromium && npx playwright test` in `e2e/`.

## License

MIT
