<script>
  import { get } from './api.js'

  let { type, onerror } = $props()

  const types = [
    { id: 'routers', label: 'Routers' },
    { id: 'services', label: 'Services' },
    { id: 'middlewares', label: 'Middlewares' },
    { id: 'entrypoints', label: 'Entrypoints' },
  ]
  const sources = [
    ['http', 'routers'], ['tcp', 'routers'], ['udp', 'routers'],
    ['http', 'services'], ['tcp', 'services'], ['udp', 'services'],
    ['http', 'middlewares'], ['tcp', 'middlewares'],
  ]
  const columns = {
    routers: ['name', 'protocol', 'status', 'rule', 'entryPoints', 'service', 'provider'],
    services: ['name', 'protocol', 'status', 'type', 'servers', 'provider'],
    middlewares: ['name', 'protocol', 'status', 'type', 'provider'],
    entrypoints: ['name', 'address'],
  }
  const mono = new Set(['name', 'rule', 'service', 'address', 'entryPoints'])
  const labels = {
    rule: 'Rule', entryPoints: 'Entrypoints', service: 'Service', middlewares: 'Middlewares', priority: 'Priority',
    provider: 'Provider', type: 'Type', address: 'Address',
  }

  let all = $state([])
  let search = $state('')
  let status = $state('')
  let provider = $state('')
  let protocol = $state('')
  let selected = $state(null)
  let error = $state('')
  let loading = $state(true)

  const active = $derived(types.find((t) => t.id === type) ?? types[0])
  const keyOf = (i) => `${i.protocol ?? ''}/${i.name}`

  async function fetchList(path) {
    try {
      return await get(`/api/traefik/${path}?per_page=1000`)
    } catch (e) {
      if (e.unauthorized) throw e
      return []
    }
  }

  async function load() {
    loading = true
    try {
      const lists = await Promise.all([...sources.map(([p, k]) => fetchList(`${p}/${k}`)), fetchList('entrypoints')])
      const rows = sources.flatMap(([p, k], n) => lists[n].map((i) => ({ ...i, protocol: p.toUpperCase(), kind: k })))
      all = [...rows, ...lists[sources.length].map((i) => ({ ...i, kind: 'entrypoints' }))]
      error = ''
    } catch (e) {
      if (e.unauthorized) return onerror(e)
      error = e.message
    } finally {
      loading = false
    }
  }
  load()

  $effect(() => {
    type
    selected = null
    search = status = provider = protocol = ''
  })

  function cell(item, col) {
    const v = item[col]
    if (col === 'servers') {
      const urls = item.loadBalancer?.servers ?? []
      const st = item.serverStatus ?? {}
      const up = urls.filter((s) => st[s.url] === 'UP').length
      return `${urls.length}${Object.keys(st).length ? ` (${up} up)` : ''}`
    }
    if (Array.isArray(v)) return v.join(', ')
    return v ?? ''
  }

  const secretKeys = new Set(['users', 'password', 'secret', 'token', 'auth', 'authorization', 'bearer'])

  function maskSecrets(obj) {
    if (obj === null || obj === undefined) return obj
    if (Array.isArray(obj)) return obj.map((x) => (typeof x === 'string' && x.includes(':') ? `${x.split(':')[0]}:******` : maskSecrets(x)))
    if (typeof obj === 'object') {
      const res = {}
      for (const [k, v] of Object.entries(obj)) {
        if (secretKeys.has(k.toLowerCase()) && typeof v === 'string') {
          res[k] = '******'
        } else if (k.toLowerCase() === 'users' && Array.isArray(v)) {
          res[k] = v.map((u) => (typeof u === 'string' && u.includes(':') ? `${u.split(':')[0]}:******` : '******'))
        } else {
          res[k] = maskSecrets(v)
        }
      }
      return res
    }
    return obj
  }

  function details(item) {
    const masked = maskSecrets(item)
    const rows = Object.keys(labels)
      .filter((k) => masked[k] !== undefined && masked[k] !== null && masked[k] !== '')
      .map((k) => [labels[k], Array.isArray(masked[k]) ? masked[k].join(', ') : String(masked[k]), mono.has(k)])
    if ('tls' in item) rows.push(['TLS', item.tls ? 'on' : 'off', false])
    return rows
  }

  const servers = (item) => {
    const st = item.serverStatus ?? {}
    return (item.loadBalancer?.servers ?? []).map((s) => ({ url: s.url ?? s.address, state: st[s.url] }))
  }

  const ofType = $derived(all.filter((i) => i.kind === active.id))
  const counts = $derived(Object.fromEntries(types.map((t) => [t.id, all.filter((i) => i.kind === t.id).length])))
  const protocols = $derived([...new Set(ofType.map((i) => i.protocol).filter(Boolean))])
  const statuses = $derived([...new Set(ofType.map((i) => i.status).filter(Boolean))])
  const providers = $derived([...new Set(ofType.map((i) => i.provider).filter(Boolean))])
  const shown = $derived(
    ofType.filter((i) => {
      if (status && i.status !== status) return false
      if (provider && i.provider !== provider) return false
      if (protocol && i.protocol !== protocol) return false
      return !search || JSON.stringify(i).toLowerCase().includes(search.toLowerCase())
    }),
  )
</script>

<div class="head">
  <h1>Traefik objects</h1>
  <span class="muted note">Read-only view of what Traefik has loaded. Edit file-provider objects under Config files.</span>
</div>

<div class="seg" role="tablist" aria-label="Object type">
  {#each types as t}
    <a href={`#/objects/${t.id}`} role="tab" aria-selected={t.id === active.id}>
      {t.label} <span class="count">{counts[t.id]}</span>
    </a>
  {/each}
</div>

<div class="bar">
  <input type="search" placeholder="Search {active.label.toLowerCase()}" bind:value={search} aria-label="Search" />
  {#if protocols.length > 1}
    <select bind:value={protocol} aria-label="Protocol">
      <option value="">All protocols</option>
      {#each protocols as p}<option>{p}</option>{/each}
    </select>
  {/if}
  {#if statuses.length}
    <select bind:value={status} aria-label="Status">
      <option value="">All statuses</option>
      {#each statuses as s}<option>{s}</option>{/each}
    </select>
  {/if}
  {#if providers.length > 1}
    <select bind:value={provider} aria-label="Provider">
      <option value="">All providers</option>
      {#each providers as p}<option>{p}</option>{/each}
    </select>
  {/if}
  <button onclick={load}>Refresh</button>
  <span class="muted note">{shown.length} of {ofType.length}</span>
</div>

{#if error}
  <p class="bad" role="alert">Cannot load: {error}</p>
{:else if loading && !all.length}
  <p class="muted">Loading…</p>
{:else}
  <div class="card scroll">
    <table>
      <thead><tr>{#each columns[active.id] as c}<th>{c === 'entryPoints' ? 'entrypoints' : c}</th>{/each}</tr></thead>
      <tbody>
        {#each shown as item (keyOf(item))}
          <tr class:sel={selected && keyOf(selected) === keyOf(item)} onclick={() => (selected = item)}>
            {#each columns[active.id] as c}
              {#if c === 'status'}
                <td><span class={item.status}>● {item.status}</span></td>
              {:else}
                <td class:mono={mono.has(c)} class:muted={c === 'provider' || c === 'protocol'}>{cell(item, c)}</td>
              {/if}
            {/each}
          </tr>
        {:else}
          <tr><td colspan={columns[active.id].length} class="muted">Nothing matches.</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

{#if selected}
  <aside class="card" aria-label="Detail">
    <div class="top">
      <div class="stack ident">
        <h2 class="mono">{selected.name}</h2>
        {#if selected.status}<span class={selected.status}>● {selected.status}</span>{/if}
      </div>
      <button class="close" aria-label="Close detail" onclick={() => (selected = null)}>×</button>
    </div>

    {#if selected.error?.length}
      <p class="bad" role="alert">{selected.error.join('; ')}</p>
    {/if}

    <dl>
      {#each details(selected) as [k, v, m]}
        <dt class="muted">{k}</dt><dd class:mono={m}>{v}</dd>
      {/each}
    </dl>

    {#if servers(selected).length}
      <div class="stack backends">
        <h3 class="muted">Backend servers</h3>
        {#each servers(selected) as s}
          <div class="server">
            <span class="mono">{s.url}</span>
            {#if s.state}<span class={s.state === 'UP' ? 'good' : 'bad'}>{s.state}</span>{/if}
          </div>
        {/each}
      </div>
    {/if}

    {#if selected.provider === 'file'}
      <a class="edit" href="#/config">Edit in config files</a>
    {/if}

    <details>
      <summary class="muted">Raw JSON</summary>
      <pre>{JSON.stringify(maskSecrets(selected), null, 2)}</pre>
    </details>
  </aside>
{/if}

<style>
  .head { display: flex; align-items: baseline; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
  .head h1 { font-size: 24px; font-weight: 600; }
  .note { font-size: 13px; }
  .seg { display: flex; flex-wrap: wrap; gap: 6px; }
  .seg a { padding: 8px 14px; border-radius: 8px; border: 1px solid var(--border); background: var(--surface); text-decoration: none; font-size: 14px; color: var(--muted); }
  .seg a:hover { background: var(--hover); }
  .seg a[aria-selected='true'] { color: var(--text); font-weight: 600; border-color: var(--accent); background: var(--hover); }
  .count { margin-left: 4px; font-size: 12px; color: var(--muted); font-weight: 400; }
  .bar { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
  .bar input[type='search'] { flex: 1 1 240px; max-width: 360px; }
  .scroll { overflow-x: auto; }
  table { border-collapse: collapse; width: 100%; font-size: 14px; }
  th { text-align: left; padding: 12px 16px; font-weight: 500; color: var(--muted); font-size: 12px; letter-spacing: 0.05em; text-transform: uppercase; }
  td { padding: 12px 16px; border-top: 1px solid var(--border); white-space: nowrap; }
  td.mono { font-size: 13px; }
  td:nth-child(4), td:nth-child(5) { white-space: normal; word-break: break-word; }
  tbody tr { cursor: pointer; }
  tbody tr:hover, tr.sel { background: var(--hover); }
  tr.sel td:first-child { box-shadow: inset 3px 0 0 var(--accent); font-weight: 500; }
  .enabled { color: var(--good); font-weight: 500; }
  .warning { color: var(--warn); font-weight: 500; }
  .disabled { color: var(--bad); font-weight: 500; }
  aside { padding: 20px; display: flex; flex-direction: column; gap: 18px; }
  .top { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
  .ident { gap: 6px; }
  .ident h2 { font-size: 18px; font-weight: 600; }
  .ident span { font-size: 13px; font-weight: 500; }
  .close { width: 36px; height: 36px; padding: 0; font-size: 18px; line-height: 1; }
  dl { margin: 0; display: grid; grid-template-columns: minmax(90px, 140px) 1fr; gap: 12px; font-size: 14px; }
  dd { margin: 0; word-break: break-word; }
  dd.mono { font-size: 13px; }
  .backends { gap: 8px; }
  .backends h3 { font-size: 13px; font-weight: 500; }
  .server { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 12px; border-radius: 8px; background: var(--code); font-size: 13px; }
  .server span:last-child { font-weight: 500; }
  .edit { height: 40px; display: flex; align-items: center; justify-content: center; border-radius: 8px; border: 1px solid var(--border); text-decoration: none; font-weight: 500; font-size: 14px; }
  .edit:hover { background: var(--hover); }
  summary { cursor: pointer; font-size: 13px; }
  pre { margin: 10px 0 0; padding: 12px; border-radius: 8px; background: var(--code); font-family: var(--mono); font-size: 12px; overflow: auto; }
</style>
