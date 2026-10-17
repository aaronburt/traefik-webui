# Traefik WebUI: Requirements

Goal: a robust, user-friendly web UI that **monitors** Traefik's status and **edits** its configuration (not read-only).

Priority tags: **[MUST]** required for v1, **[SHOULD]** strongly desired, **[COULD]** later.
Each item is written so it can be verified (acceptance criteria are in section 14).

---

## 1. Scope and assumptions

- Targets Traefik **v3.x only** (decided 2026-10-17; v2.x is not supported in v1).
- The UI is a separate service/container that talks to Traefik; Traefik itself is not modified.
- Single Traefik instance in v1 ([MUST]); multiple instances/environments [SHOULD].
- Primary users: homelab owners and small-team operators/admins.
- **Decided (2026-10-17):** v1 edits the **file provider** only; Docker/Kubernetes-sourced config is shown read-only.
- **Decided (2026-10-17):** tech stack is **Go** (backend) + **Svelte** (frontend, built as a static SPA and embedded in the Go binary). Changed from React the same day.
- **Decided (2026-10-17):** v1 is **single-user**: one admin login, no roles. Section 5 role-based items are deferred (kept as a later option); authentication itself stays [MUST].
- **Decided (2026-10-17):** the UI never restarts Traefik. Changes to static config show a "restart required" warning only; the user restarts Traefik themselves.
- Open question (partly answered above): which config providers are in use (file, Docker, Kubernetes CRD/Ingress, Consul, etc.)? This decides what is editable (see section 4).

## 2. Architecture

- **[MUST]** Backend service that proxies/aggregates Traefik's API so the browser never needs direct access to Traefik's API or admin ports.
- **[MUST]** Frontend SPA (framework TBD) served by the backend; responsive down to tablet width.
- **[MUST]** Runs as a Docker container; published image, `docker-compose.yml` example, and a Traefik-label example for exposing the UI itself behind Traefik.
- **[SHOULD]** Single binary/image with no external database required (embedded storage, e.g. SQLite) for users, audit log, and history.
- **[SHOULD]** Stateless API where possible; all state in a mounted volume.
- **[COULD]** Helm chart / Kubernetes manifests.

## 3. Monitoring (read side)

Data source: Traefik API (`/api/overview`, `/api/rawdata`, `/api/http|tcp|udp/routers|services|middlewares`, `/api/entrypoints`, `/api/version`) and the Prometheus metrics endpoint.

- **[MUST]** Dashboard overview: Traefik version, uptime/health, counts of routers/services/middlewares per protocol (HTTP/TCP/UDP), and counts of warnings/errors.
- **[MUST]** Lists with search, filter (status, provider, protocol, entrypoint), sort, and pagination for: routers, services, middlewares, entrypoints.
- **[MUST]** Detail views showing rule, priority, entrypoints, TLS settings, attached middlewares, target service, load-balancer servers and their health, provider, and status/error messages.
- **[MUST]** Clear status indicators (enabled / warning / disabled) with the Traefik-reported error text.
- **[MUST]** Show configuration errors and conflicts (duplicate routers, missing service, invalid middleware reference).
- **[SHOULD]** Relationship graph/visualization: entrypoint → router → middleware chain → service → servers.
- **[SHOULD]** Server (backend) health view per service (up/down from health checks).
- **[COULD]** (Deferred past v1 by decision on 2026-10-17: v1 shows live status only, no metrics charts.) Metrics page from Prometheus metrics: request rate, status-code breakdown (2xx/4xx/5xx), latency percentiles, open connections, per router/service/entrypoint, with selectable time range.
- **[SHOULD]** Auto-refresh with configurable interval and a manual refresh; visible "last updated" time.
- **[SHOULD]** TLS/certificate view: domains, issuer, expiry date, ACME resolver status, with warnings for certificates nearing expiry (configurable threshold).
- **[SHOULD]** Access log viewer (tail, filter by status/router/client IP/path) when access logs are available to the UI.
- **[SHOULD]** Traefik application log viewer with level filter.
- **[COULD]** Historical charts persisted by the UI (beyond what Prometheus retains).
- **[COULD]** Alerting (cert expiry, service down, 5xx spike) via webhook/email.

## 4. Configuration editing (write side)

Traefik has two configuration layers, and the UI must be explicit about which it edits:

- **Dynamic configuration** (routers, services, middlewares, TLS options/stores, serversTransports): hot-reloaded.
- **Static configuration** (entrypoints, providers, certificate resolvers, logging, API, metrics): requires a Traefik restart.

Requirements:

- **[MUST]** Edit dynamic configuration through the **file provider** (YAML/TOML files in a watched directory the UI can write to). This is the safest universal write path and the baseline for v1.
- **[MUST]** Full CRUD for HTTP routers, services, and middlewares; TCP routers/services/middlewares; UDP routers/services; TLS options, stores, and certificates; serversTransports.
- **[MUST]** Two editing modes: guided forms (typed fields, dropdowns of existing services/middlewares/entrypoints/resolvers) and a raw YAML/TOML editor with syntax highlighting. Switching between modes must not lose data.
- **[MUST]** Support all built-in middleware types with typed forms (addPrefix, basicAuth, forwardAuth, headers, rateLimit, redirectScheme, redirectRegex, stripPrefix, replacePath, compress, circuitBreaker, retry, ipAllowList, chain, buffering, inFlightReq, errors, etc.), driven by a schema so new middlewares are cheap to add.
- **[MUST]** Rule builder/validator for router `rule` expressions (Host, PathPrefix, Headers, ClientIP, combinators) with live syntax checking.
- **[MUST]** Validation before save: schema validation, reference checks (router → service/middleware exists), duplicate-name detection, and rule syntax checks. Block saving invalid config with clear, field-level error messages.
- **[MUST]** Diff preview (before vs after) and an explicit confirm step before applying any change.
- **[MUST]** Atomic writes (write temp file, validate, rename) so Traefik never reads a half-written file.
- **[MUST]** Automatic backup/versioning of every config file before it is changed; one-click rollback to any prior version.
- **[MUST]** After apply, verify Traefik accepted the change (poll the API for the new object / errors) and report success or the Traefik error back to the user; auto-rollback option on failure.
- **[MUST]** Read-only display (with "managed by <provider>" badge) for objects that come from other providers (Docker labels, Kubernetes, etc.) that the UI cannot safely edit. The UI never silently edits what it does not own.
- **[SHOULD]** Static configuration viewer ([MUST] read-only) and editor ([SHOULD]) with a clear "requires restart" warning, plus an optional restart action via a configurable mechanism (Docker socket/API, signal, or user-supplied command/webhook).
- **[SHOULD]** Clone/duplicate an object; bulk enable/disable; import/export of config (YAML) for backup and migration.
- **[SHOULD]** Templates/presets for common setups (reverse proxy a service with HTTPS + redirect, basic-auth-protected route, www redirect, security-headers set).
- **[SHOULD]** Dry-run / "test this rule" tool: enter a host/path/method and see which router would match.
- **[SHOULD]** Certificate resolver and ACME settings management (static config) with clear restart notice; upload custom certificates (dynamic).
- **[COULD]** Docker provider integration: view labels per container and offer a label editor that outputs labels to copy (UI does not edit running containers).
- **[COULD]** Kubernetes CRD editing (IngressRoute, Middleware, etc.) via the Kubernetes API.
- **[COULD]** Git-backed config (commit each change, optional push to a remote).

## 5. Authentication and authorization

- **[MUST]** The UI must not be usable without authentication by default (no anonymous write access).
- **[MUST]** Local accounts with strong password hashing (argon2id/bcrypt), forced setup of the first admin on first run, password change.
- **[MUST]** Role-based access control: at minimum **Admin** (full), **Editor** (edit dynamic config), **Viewer** (read-only).
- **[MUST]** Secure sessions: HttpOnly, Secure, SameSite cookies, CSRF protection, session expiry and logout, login rate limiting/lockout.
- **[SHOULD]** OIDC/OAuth2 single sign-on (and compatibility with forwardAuth setups such as Authelia/Authentik) with group-to-role mapping.
- **[SHOULD]** TOTP two-factor authentication for local accounts.
- **[SHOULD]** API tokens for automation, scoped by role, revocable.
- **[COULD]** Per-resource permissions (e.g. an editor may only touch certain routers).

## 6. Security

- **[MUST]** Secrets handling: values such as basic-auth hashes, API tokens, and ACME credentials are masked in the UI and never written to logs; sensitive fields are write-only after saving.
- **[MUST]** The UI's connection credentials to Traefik/Docker come from environment variables or secret files, never stored in plain text in the browser.
- **[MUST]** Input validation and output encoding throughout (no XSS, injection, or path traversal); file writes are restricted to the configured config directory.
- **[MUST]** Security headers (CSP, HSTS when behind TLS, X-Content-Type-Options, frame protections) on the UI itself.
- **[MUST]** Least privilege: if the Docker socket is used, document the risk and support a socket proxy; run as non-root; read-only root filesystem where possible.
- **[MUST]** Audit log of every write and sign-in event (who, when, what changed, diff, source IP), viewable and exportable, tamper-evident where feasible.
- **[SHOULD]** Dependency vulnerability scanning and image scanning in CI.
- **[SHOULD]** Warning banner when risky settings are detected (insecure API enabled, dashboard exposed without auth, default entrypoint on HTTP only).

## 7. Usability

- **[MUST]** Navigation by object type (Dashboard, Routers, Services, Middlewares, Entrypoints, TLS/Certificates, Logs, Settings) with global search.
- **[MUST]** Light and dark themes (follow system by default).
- **[MUST]** Helpful inline documentation: tooltips and links to the relevant Traefik docs for each field.
- **[MUST]** Meaningful, human-readable error messages; no raw stack traces in the UI.
- **[MUST]** Unsaved-changes protection and confirmation dialogs for destructive actions (delete, rollback, restart).
- **[SHOULD]** Keyboard shortcuts and a command palette.
- **[SHOULD]** Accessibility to WCAG 2.1 AA (keyboard navigation, contrast, labels, screen reader support).
- **[SHOULD]** Mobile-friendly layout for the monitoring views.
- **[SHOULD]** Internationalization-ready (string externalization), English first.
- **[COULD]** Onboarding wizard that detects the Traefik setup and guides the first connection.

## 8. Connectivity and deployment configuration

- **[MUST]** Configurable via environment variables/config file: Traefik API URL, optional basic-auth/TLS settings for that API, config directory path, metrics URL, listen address/port, base path (for serving under a sub-path).
- **[MUST]** Startup self-check with clear diagnostics (cannot reach API, config dir not writable, version unsupported) shown in the UI and logs.
- **[MUST]** Health endpoint for the UI itself (`/healthz`) for container orchestration.
- **[SHOULD]** Support for Traefik API secured with its own auth/TLS, and for connecting over a private Docker network without exposing the API.
- **[SHOULD]** Graceful handling of Traefik being temporarily unreachable (stale-data indicator, automatic reconnect).
- **[COULD]** Multiple Traefik instances/environments with a switcher.

## 9. Non-functional requirements

- **Performance:** initial dashboard load under 2 s on a typical LAN with up to 500 routers; list views remain responsive (virtualized/paginated) at 5,000 objects. Idle backend memory under about 100 MB.
- **Reliability:** UI failure must never take down Traefik or corrupt its configuration; all writes are atomic and reversible.
- **Compatibility:** latest two versions of Chrome, Firefox, Safari, Edge; images for amd64 and arm64 (Raspberry Pi/home servers).
- **Observability of the UI itself:** structured logs, configurable log level, optional Prometheus metrics for the UI.
- **Maintainability:** typed codebase, linting, formatted code, clear module boundaries; the middleware/config forms are generated from a schema to track Traefik changes.
- **Documentation:** README with quick start, configuration reference, security hardening guide, and contribution guide.
- **Licensing:** MIT (decided 2026-10-17); include a LICENSE file in the repo.

## 10. API (UI backend)

- **[MUST]** Documented REST API (OpenAPI spec) used by the frontend; all write endpoints authenticated and role-checked.
- **[MUST]** Endpoints for: read of all Traefik objects (normalized), config CRUD, validate (dry-run), diff, apply, backups list/restore, audit log, auth/session, and health.
- **[SHOULD]** Server-sent events or WebSocket for live updates (status changes, log tail).
- **[SHOULD]** Pagination, filtering, and sorting handled server-side.

## 11. Data and persistence

- **[MUST]** Config backups stored on a mounted volume with retention policy (count and/or age, configurable).
- **[MUST]** User, role, session, and audit data persisted across restarts.
- **[SHOULD]** Export/import of UI settings and users.
- **[SHOULD]** Safe migrations between UI versions.

## 12. Testing and quality

- **[MUST]** Unit tests for validation, rule parsing, and config generation (including round-trip YAML preservation).
- **[MUST]** Integration tests against a real Traefik container in CI (start Traefik, apply changes through the UI backend, verify Traefik accepts them).
- **[SHOULD]** End-to-end browser tests (Playwright) for the key flows: login, create route, edit middleware, rollback.
- **[SHOULD]** Contract tests against supported Traefik versions (matrix in CI).
- **[MUST]** CI pipeline: lint, type-check, tests, build, container image publish, release tagging.

## 13. Phased delivery (proposed)

1. **Phase 1, Monitor:** connect to the Traefik API; dashboard, lists, detail views, error display, auth (local + roles), Docker image.
2. **Phase 2, Edit dynamic config:** file-provider CRUD for HTTP routers/services/middlewares, validation, diff, atomic apply, backups and rollback, audit log.
3. **Phase 3, Breadth:** TCP/UDP, TLS and certificates, static config viewer/editor, templates, raw editor polish, import/export.
4. **Phase 4, Insight and integration:** Prometheus metrics charts, log viewers, OIDC, alerting, relationship graph.
5. **Phase 5, Ecosystem:** Docker label tooling, Kubernetes CRDs, multi-instance, Git-backed config.

## 14. Acceptance criteria for v1 (Phases 1 and 2)

- A new user can start the container with the compose example, create the first admin, and see live Traefik status within 5 minutes.
- Every router, service, and middleware reported by Traefik appears in the UI with correct status and error text.
- A user can create a new HTTP route (router + service + middleware chain) using forms only, preview the diff, apply it, and reach the backend through Traefik without restarting Traefik.
- An invalid change (bad rule, missing service reference) is rejected before reaching disk with a clear message.
- A change that Traefik rejects after apply is detected and can be rolled back in one click (or automatically).
- Objects owned by other providers are shown as read-only and cannot be modified.
- Viewer-role users cannot perform any write; unauthenticated requests are refused.
- Every write appears in the audit log with a diff.
- No secrets appear in logs, API responses, or the browser after being saved.

## 15. Open questions

1. Which Traefik version(s) and providers do you run (file, Docker, Kubernetes)?
2. Should the UI only manage file-provider config in v1, or also Docker labels/Kubernetes?
3. Preferred tech stack (backend language and frontend framework)?
4. Single-user homelab tool, or multi-user with SSO and roles from the start?
5. Is restarting Traefik from the UI wanted, and via what mechanism (Docker socket, signal, command)?
6. Is Prometheus already available for metrics, or should the UI collect its own history?
7. License and whether this will be published for others to use.
