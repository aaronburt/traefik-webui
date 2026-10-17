<script>
  import { get } from './api.js'

  // kind is a path under the Traefik API, e.g. "http/routers" or "entrypoints".
  let { kind, onerror } = $props()

  let items = $state([])
  let search = $state('')
  let status = $state('')
  let provider = $state('')
  let selected = $state(null)
  let error = $state('')
  let loading = $state(true)

  const columns = {
    'http/routers': ['name', 'status', 'rule', 'entryPoints', 'service', 'provider'],
    'http/services': ['name', 'status', 'type', 'servers', 'provider'],
    'http/middlewares': ['name', 'status', 'type', 'provider'],
    'tcp/routers': ['name', 'status', 'rule', 'entryPoints', 'service', 'provider'],
    'tcp/services': ['name', 'status', 'type', 'provider'],
    'udp/routers': ['name', 'status', 'entryPoints', 'service', 'provider'],
    'udp/services': ['name', 'status', 'type', 'provider'],
    entrypoints: ['name', 'address'],
  }
  const monoColumns = ['name', 'rule', 'service', 'entryPoints', 'address']

  async function load() {
    loading = true
    try {
      items = await get(`/api/traefik/${kind}?per_page=1000`)
      error = ''
    } catch (e) {
      if (e.unauthorized) return onerror(e)
      error = e.message
    } finally {
      loading = false
    }
  }
  $effect(() => {
    kind
    selected = null
    load()
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

  function statusClass(s) {
    return { enabled: 'good', warning: 'warn', disabled: 'bad' }[s] ?? 'muted'
  }

  const statuses = $derived([...new Set(items.map((i) => i.status).filter(Boolean))])
  const providers = $derived([...new Set(items.map((i) => i.provider).filter(Boolean))])
  const shown = $derived(
    items.filter((i) => {
      if (status && i.status !== status) return false
      if (provider && i.provider !== provider) return false
      return !search || JSON.stringify(i).toLowerCase().includes(search.toLowerCase())
    }),
  )
</script>

<div class="toolbar">
  <input class="search" type="search" placeholder="Search…" bind:value={search} aria-label="Search" />
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
  <span class="muted count">{shown.length} of {items.length}</span>
</div>

{#if error}
  <p class="bad" role="alert">Cannot load: {error}</p>
{:else if loading && !items.length}
  <p class="muted">Loading…</p>
{:else}
  <div class="card table-card">
    <div class="scroll">
      <table>
        <thead>
          <tr>{#each columns[kind] as c}<th scope="col">{c}</th>{/each}</tr>
        </thead>
        <tbody>
          {#each shown as item (item.name)}
            <tr class:sel={selected?.name === item.name} onclick={() => (selected = item)}>
              {#each columns[kind] as c}
                <td class:mono={monoColumns.includes(c)} class:wrap={c === 'rule'}>
                  {#if c === 'status'}
                    <span class={statusClass(item.status)}><span class="dot"></span>{item.status ?? 'unknown'}</span>
                  {:else}
                    {cell(item, c)}
                  {/if}
                </td>
              {/each}
            </tr>
          {:else}
            <tr><td colspan={columns[kind].length} class="muted">Nothing matches.</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}

{#if selected}
  <section class="card detail" aria-label={`Details for ${selected.name}`}>
    <div class="detail-head">
      <h2 class="mono">{selected.name}</h2>
      <button onclick={() => (selected = null)} aria-label="Close details">Close</button>
    </div>
    {#if selected.error?.length}
      <p class="bad" role="alert">{selected.error.join('; ')}</p>
    {/if}
    <pre>{JSON.stringify(selected, null, 2)}</pre>
  </section>
{/if}

<style>
  .toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px;
    margin-bottom: 16px;
  }
  .search {
    flex: 1 1 240px;
    max-width: 360px;
  }
  .count {
    font-size: 13px;
  }
  .table-card {
    margin-bottom: 20px;
    padding: 0;
    overflow: hidden;
  }
  .scroll {
    overflow-x: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 14px;
  }
  th {
    padding: 12px 16px;
    font-size: 12px;
    font-weight: 500;
    letter-spacing: 0.05em;
    text-align: left;
    text-transform: uppercase;
    color: var(--muted);
  }
  td {
    padding: 12px 16px;
    border-top: 1px solid var(--border);
    white-space: nowrap;
    vertical-align: top;
  }
  td.wrap {
    white-space: normal;
    word-break: break-word;
  }
  tbody tr {
    cursor: pointer;
  }
  tbody tr:hover {
    background: var(--hover);
  }
  tbody tr.sel {
    background: var(--hover);
    box-shadow: inset 3px 0 0 var(--accent);
  }
  .detail {
    margin-top: 4px;
  }
  .detail-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
  }
  .detail h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
  }
  pre {
    margin: 0;
    padding: 14px;
    border-radius: 8px;
    background: var(--code);
    font-family: 'IBM Plex Mono', ui-monospace, monospace;
    font-size: 12px;
    overflow: auto;
  }
</style>
