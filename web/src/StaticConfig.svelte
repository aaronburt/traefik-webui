<script>
  import { get } from './api.js'

  let { onerror } = $props()
  let version = $state(null)
  let overview = $state(null)
  let entrypoints = $state([])
  let error = $state('')

  async function load() {
    try {
      ;[version, overview, entrypoints] = await Promise.all([
        get('/api/traefik/version'),
        get('/api/traefik/overview'),
        get('/api/traefik/entrypoints'),
      ])
      error = ''
    } catch (e) {
      if (e.unauthorized) return onerror(e)
      error = e.message
    }
  }
  load()

  const features = $derived(Object.entries(overview?.features ?? {}))
  const show = (v) => (v === false || v === '' || v == null ? 'off' : v === true ? 'on' : v)
  const timeout = (e) => e.transport?.respondingTimeouts ?? {}
</script>

<div class="head">
  <h1>Static config</h1>
  <button onclick={load}>Refresh</button>
</div>

<p class="muted intro">
  The settings Traefik was started with. They are read-only here: change them in <span class="mono">traefik.yml</span>, command-line flags or environment variables, then restart Traefik.
</p>

{#if error}
  <p class="bad" role="alert">Cannot reach Traefik: {error}</p>
{:else if !overview}
  <p class="muted">Loading…</p>
{:else}
  <section class="grid">
    <div class="card tile">
      <h2>Instance</h2>
      <div class="stack rows">
        <div class="row"><span class="muted">Version</span><span class="mono">{version?.Version}</span></div>
        {#if version?.Codename}<div class="row"><span class="muted">Codename</span><span>{version.Codename}</span></div>{/if}
        {#if version?.startDate}<div class="row"><span class="muted">Started</span><span>{new Date(version.startDate).toLocaleString()}</span></div>{/if}
      </div>
    </div>
    <div class="card tile">
      <h2>Providers</h2>
      <div class="pills">
        {#each overview.providers ?? [] as p}<span class="pill">{p}</span>{:else}<span class="muted">none</span>{/each}
      </div>
    </div>
    <div class="card tile">
      <h2>Features</h2>
      <div class="stack rows">
        {#each features as [k, v]}
          <div class="row"><span class="muted">{k}</span><span>{show(v)}</span></div>
        {:else}
          <span class="muted">none</span>
        {/each}
      </div>
    </div>
  </section>

  <h2 class="section">Entrypoints <span class="muted count">{entrypoints.length}</span></h2>
  <section class="eps">
    {#each entrypoints as e (e.name)}
      <div class="card tile">
        <div class="row">
          <h3 class="mono">{e.name}</h3>
          <span class="mono addr">{e.address}</span>
        </div>
        <div class="stack rows small">
          {#if e.http?.redirections?.entryPoint}
            <div class="row"><span class="muted">Redirects to</span><span class="mono">{e.http.redirections.entryPoint.to}</span></div>
          {/if}
          {#if e.http?.tls}
            <div class="row"><span class="muted">TLS</span><span>on{e.http.tls.certResolver ? ` · resolver ${e.http.tls.certResolver}` : ''}</span></div>
          {/if}
          {#if timeout(e).readTimeout}<div class="row"><span class="muted">Read timeout</span><span class="mono">{timeout(e).readTimeout}</span></div>{/if}
          {#if timeout(e).idleTimeout}<div class="row"><span class="muted">Idle timeout</span><span class="mono">{timeout(e).idleTimeout}</span></div>{/if}
          {#if e.http2?.maxConcurrentStreams}<div class="row"><span class="muted">HTTP/2 streams</span><span class="mono">{e.http2.maxConcurrentStreams}</span></div>{/if}
        </div>
        <details>
          <summary class="muted">Raw JSON</summary>
          <pre>{JSON.stringify(e, null, 2)}</pre>
        </details>
      </div>
    {:else}
      <p class="muted">No entrypoints reported.</p>
    {/each}
  </section>

  <p class="muted note">
    Traefik's API does not expose everything from the static file, such as certificate resolvers, log settings and API options. Check the file or flags you started Traefik with for those.
  </p>
{/if}

<style>
  .head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
  .head h1 { font-size: 24px; font-weight: 600; }
  .intro { font-size: 14px; line-height: 1.5; max-width: 720px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 16px; }
  .eps { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 16px; }
  .tile { padding: 20px; display: flex; flex-direction: column; gap: 14px; }
  .tile h2 { font-size: 16px; font-weight: 600; }
  .tile h3 { font-size: 15px; font-weight: 600; }
  .section { font-size: 18px; font-weight: 600; }
  .count { font-size: 14px; font-weight: 400; }
  .rows { gap: 10px; font-size: 14px; }
  .rows.small { gap: 8px; }
  .row { display: flex; justify-content: space-between; align-items: baseline; gap: 12px; }
  .addr { font-size: 13px; }
  .pills { display: flex; flex-wrap: wrap; gap: 8px; }
  .note { font-size: 13px; line-height: 1.5; max-width: 720px; }
  summary { cursor: pointer; font-size: 13px; }
  pre { margin: 10px 0 0; padding: 12px; border-radius: 8px; background: var(--code); font-family: var(--mono); font-size: 12px; overflow: auto; }
</style>
