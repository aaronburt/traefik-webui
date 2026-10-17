<script>
  import { get } from './api.js'

  let { onerror } = $props()
  const pageIds = ['routers', 'services', 'middlewares', 'tcp-routers', 'tcp-services', 'udp-routers', 'udp-services']
  const protocols = ['http', 'tcp', 'udp']
  const kinds = ['routers', 'services', 'middlewares']

  let overview = $state(null)
  let error = $state('')

  async function load() {
    try {
      overview = await get('/api/traefik/overview')
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
      .map((k) => ({ kind: k, id: `${p === 'http' ? '' : p + '-'}${k}`, ...o[p][k] }))

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
    {#each protocols as p}
      <section class="card">
        <h2>{p.toUpperCase()}</h2>
        <ul class="rows">
          {#each rows(overview, p) as r}
            <li>
              {#if pageIds.includes(r.id)}<a href={`#/${r.id}`}>{r.kind}</a>{:else}<span>{r.kind}</span>{/if}
              <span class="counts mono">
                {r.total}
                {#if r.errors}<span class="bad"> · {r.errors} err</span>{/if}
                {#if r.warnings}<span class="warn"> · {r.warnings} warn</span>{/if}
              </span>
            </li>
          {/each}
        </ul>
      </section>
    {/each}
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
