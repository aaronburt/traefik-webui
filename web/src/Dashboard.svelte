<script>
  import { get } from './api.js'

  let { onerror } = $props()
  const protocols = ['http', 'tcp', 'udp']
  const kinds = ['routers', 'services', 'middlewares']
  // Pages that list each kind. Routers and services cover every protocol; middlewares only HTTP.
  const pageProtocols = { routers: protocols, services: protocols, middlewares: ['http'] }
  // HTTP gets its own card; TCP and UDP share one.
  const cards = [
    { title: 'HTTP', protocols: ['http'] },
    { title: 'TCP & UDP', protocols: ['tcp', 'udp'] },
  ]

  let overview = $state(null)
  let entrypoints = $state([])
  let error = $state('')

  async function load() {
    try {
      const [o, eps] = await Promise.all([get('/api/traefik/overview'), get('/api/traefik/entrypoints')])
      overview = o
      entrypoints = eps
      error = ''
    } catch (e) {
      if (e.unauthorized) return onerror(e)
      error = e.message
    }
  }
  load()
  const timer = setInterval(load, 10000)
  $effect(() => () => clearInterval(timer))

  // Objects of one protocol, skipping kinds that protocol does not have.
  const rows = (o, p) =>
    kinds
      .filter((k) => o[p]?.[k] && !(p === 'udp' && k === 'middlewares'))
      .map((k) => ({ ...o[p][k], kind: k, protocol: p }))

  // Link to the Objects tab on this kind, filtered to HTTP or to TCP and UDP.
  const href = (r) => {
    if (!pageProtocols[r.kind]?.includes(r.protocol)) return ''
    return `#/objects?kind=${r.kind}&protocol=${r.protocol === 'http' ? 'http' : 'tcp-udp'}`
  }
  const label = (r) => (r.protocol === 'http' ? r.kind : `${r.protocol.toUpperCase()} ${r.kind}`)

  const errorCount = (o) => protocols.reduce((n, p) => n + kinds.reduce((m, k) => m + (o[p]?.[k]?.errors ?? 0), 0), 0)
</script>

{#if error}
  <p class="bad" role="alert">Cannot reach Traefik: {error}</p>
{:else if !overview}
  <p class="muted">Loading…</p>
{:else}
  {@const errors = errorCount(overview)}
  <p class="muted note">Refreshes every 10 seconds.</p>
  {#if errors}
    <section class="alert" role="alert">
      <strong class="bad">{errors} {errors === 1 ? 'object has' : 'objects have'} errors</strong>
      <span class="muted">Open the router, service or middleware to see Traefik's message.</span>
    </section>
  {/if}
  <div class="grid">
    {#each cards as card}
      <section class="card">
        <h2>{card.title}</h2>
        <ul class="rows">
          {#each card.protocols as p}
            {#each rows(overview, p) as r}
              <li>
                {#if href(r)}<a href={href(r)}>{label(r)}</a>{:else}<span>{label(r)}</span>{/if}
                <span class="counts mono">
                  {r.total}
                  {#if r.errors}<span class="bad"> · {r.errors} err</span>{/if}
                  {#if r.warnings}<span class="warn"> · {r.warnings} warn</span>{/if}
                </span>
              </li>
            {/each}
          {/each}
        </ul>
      </section>
    {/each}
    <section class="card">
      <h2>Entrypoints</h2>
      <ul class="rows">
        {#each entrypoints as e}
          <li><span>{e.name}</span><span class="mono">{e.address}</span></li>
        {:else}
          <li><span class="muted">None configured.</span></li>
        {/each}
      </ul>
    </section>
    <section class="card">
      <h2>Features</h2>
      <ul class="rows">
        {#each Object.entries(overview.features ?? {}) as [k, v]}
          <li><span class="muted">{k}</span><span class="mono">{v || 'off'}</span></li>
        {/each}
      </ul>
      <h2 class="sub">Providers</h2>
      <p class="chips">
        {#each overview.providers ?? [] as pr}
          <span class="chip">{pr}</span>
        {:else}
          <span class="muted">none</span>
        {/each}
      </p>
    </section>
  </div>
{/if}

<style>
  .note {
    margin: 0 0 20px;
    font-size: 14px;
  }
  .alert {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 20px;
    padding: 14px 18px;
    border: 1px solid var(--bad);
    border-radius: 12px;
    background: var(--surface);
    font-size: 14px;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 16px;
  }
  h2 {
    margin: 0 0 14px;
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--muted);
  }
  h2.sub {
    margin-top: 22px;
  }
  .rows {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .rows li {
    display: flex;
    justify-content: space-between;
    gap: 12px;
  }
  .counts {
    white-space: nowrap;
  }
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin: 0;
  }
  .chip {
    padding: 4px 10px;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--hover);
    font-size: 13px;
  }
</style>
