<script>
  import { get } from './api.js'

  let { onerror } = $props()
    let overview = $state(null)
  let routers = $state([])
  let files = $state([])
  let updated = $state(null)
  let error = $state('')

  async function load() {
    try {
      ;[overview, routers, files] = await Promise.all([
        get('/api/traefik/overview'),
        get('/api/traefik/http/routers?per_page=1000').catch(() => []),
        get('/api/config/files').catch(() => []),
      ])
      updated = new Date()
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

  const broken = $derived(routers.filter((r) => r.error?.length || r.status === 'disabled'))
  const recent = $derived([...files].sort((a, b) => new Date(b.modTime) - new Date(a.modTime)).slice(0, 5))
  const features = $derived(Object.entries(overview?.features ?? {}))
</script>

<div class="page-head">
  <h1>Dashboard</h1>
  {#if updated}<span class="muted note">Updated {updated.toLocaleTimeString()} · refreshes every 10 s</span>{/if}
</div>

{#if error}
  <p class="bad" role="alert">Cannot reach Traefik: {error}</p>
{:else if !overview}
  <p class="muted">Loading…</p>
{:else}
  {#if broken.length}
    <section class="banner bad" aria-label="Attention">
      <svg width="20" height="20" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" stroke-width="1.5" /><path d="M8 4.5v4M8 10.8v.4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" /></svg>
      <div class="stack msg">
        <strong>{broken.length} {broken.length === 1 ? 'router has' : 'routers have'} an error</strong>
        <span class="muted"><span class="mono name">{broken[0].name}</span> is {broken[0].status}{broken[0].error?.length ? `: ${broken[0].error.join('; ')}` : ''}</span>
      </div>
      <a class="link" href="#/objects/routers">View routers</a>
    </section>
  {/if}

  <section class="totals" aria-label="Totals">
    {#each cards(overview) as c}
      <div class="card tile">
        <h2 class="eyebrow">{c.protocol}</h2>
        <div class="stack rows">
          {#each c.items as i}
                        <div class="row">
              <a href={`#/objects/${i.kind}`}>{i.kind}</a>
              <span class="mono num">
                {i.total}
                {#if i.warnings}<small class="warn">{i.warnings} warning{i.warnings === 1 ? '' : 's'}</small>{/if}
                {#if i.errors}<small class="bad">{i.errors} error{i.errors === 1 ? '' : 's'}</small>{/if}
              </span>
            </div>
          {/each}
        </div>
      </div>
    {/each}
  </section>

  <section class="info">
    <div class="card tile">
      <h2>Providers</h2>
      <div class="pills">
        {#each overview.providers ?? [] as p}<span class="pill">{p}</span>{:else}<span class="muted">none</span>{/each}
      </div>
      <p class="muted help">Only objects from the file provider can be edited here. Other providers are shown read-only.</p>
    </div>
    <div class="card tile">
      <h2>Features</h2>
      <div class="stack rows small">
        {#each features as [k, v]}
          <div class="row"><span class="muted">{k}</span><span>{v || 'off'}</span></div>
        {:else}
          <span class="muted">none</span>
        {/each}
      </div>
    </div>
    <div class="card tile">
      <h2>Recent config changes</h2>
      <div class="stack rows small">
        {#each recent as f}
          <div class="row"><span class="mono">{f.name}</span><span class="muted">{new Date(f.modTime).toLocaleString()}</span></div>
        {:else}
          <span class="muted">No config files yet.</span>
        {/each}
      </div>
      <a class="link" href="#/config">Open config files</a>
    </div>
  </section>
{/if}

<style>
  .note { font-size: 13px; }
  .msg { gap: 4px; font-size: 14px; }
  .name { color: var(--text); }
  .link { margin-left: auto; font-size: 14px; font-weight: 500; color: var(--accent); white-space: nowrap; }
  .totals { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 16px; }
  .info { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 16px; }
  .tile { padding: 20px; display: flex; flex-direction: column; gap: 14px; }
  .tile h2 { font-size: 16px; font-weight: 600; }
  .tile h2.eyebrow { font-size: 13px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--muted); font-weight: 600; }
  .rows { gap: 10px; font-size: 15px; }
  .rows.small { font-size: 14px; gap: 8px; }
  .row { display: flex; justify-content: space-between; gap: 12px; }
  .row a { color: var(--accent); text-transform: capitalize; }
  .row span:first-child { text-transform: none; }
  .num { font-weight: 500; }
  .num small { font-size: 12px; margin-left: 6px; }
  .pills { display: flex; flex-wrap: wrap; gap: 8px; }
  .pill { padding: 4px 10px; border-radius: 999px; background: var(--hover); border: 1px solid var(--border); font-size: 13px; }
  .help { font-size: 13px; line-height: 1.5; }
  .tile > .link { margin-left: 0; }
</style>
