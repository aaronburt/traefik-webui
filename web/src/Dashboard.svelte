<script>
  import { get } from './api.js'

  let { onerror } = $props()
  const pageIds = ['routers', 'services', 'middlewares', 'tcp-routers', 'tcp-services', 'udp-routers', 'udp-services']
  let overview = $state(null)
  let version = $state(null)
  let error = $state('')

  async function load() {
    try {
      ;[overview, version] = await Promise.all([get('/api/traefik/overview'), get('/api/traefik/version')])
      error = ''
    } catch (e) {
      if (e.unauthorized) return onerror(e)
      error = e.message
    }
  }
  load()
  const timer = setInterval(load, 10000)
  $effect(() => () => clearInterval(timer))

  const cards = (o) =>
    ['http', 'tcp', 'udp'].map((p) => ({
      protocol: p.toUpperCase(),
      items: ['routers', 'services', 'middlewares']
        .filter((k) => o[p]?.[k] && !(p === 'udp' && k === 'middlewares'))
        .map((k) => ({ kind: k, ...o[p][k] })),
    }))
</script>

{#if error}
  <p class="error" role="alert">Cannot reach Traefik: {error}</p>
{:else if !overview}
  <p>Loading…</p>
{:else}
  <p>Traefik <strong>{version?.Version}</strong>{#if version?.Codename}{' '}({version.Codename}){/if}. Refreshes every 10 seconds.</p>
  <div class="grid">
    {#each cards(overview) as c}
      <section>
        <h2>{c.protocol}</h2>
        {#each c.items as i}
          {@const id = `${c.protocol === 'HTTP' ? '' : c.protocol.toLowerCase() + '-'}${i.kind}`}
          <p>
            {#if pageIds.includes(id)}<a href={`#/${id}`}>{i.kind}</a>{:else}{i.kind}{/if}:
            {i.total}
            {#if i.warnings}<span class="warn">{i.warnings} warnings</span>{/if}
            {#if i.errors}<span class="bad">{i.errors} errors</span>{/if}
          </p>
        {/each}
      </section>
    {/each}
    <section>
      <h2>Features</h2>
      <p>{(overview.features && Object.entries(overview.features).map(([k, v]) => `${k}: ${v || 'off'}`).join(', ')) || 'none'}</p>
      <p>Providers: {(overview.providers ?? []).join(', ') || 'none'}</p>
    </section>
  </div>
{/if}

<style>
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr)); gap: 1rem; }
  section { border: 1px solid var(--border); border-radius: 8px; padding: 0 1rem; }
  .warn { color: var(--warn); margin-left: 0.5rem; }
  .bad, .error { color: var(--bad); margin-left: 0.5rem; }
</style>
