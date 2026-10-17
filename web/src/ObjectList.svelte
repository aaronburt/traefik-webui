<script>
  import { get } from './api.js'

  // kinds are paths under the Traefik API, e.g. ["http/routers", "tcp/routers"].
  let { kinds, columns, initialProtocol = '', onerror } = $props()

  let items = $state([])
  let search = $state('')
  let status = $state('')
  let protocol = $state('')
  let provider = $state('')
  let selected = $state(null)
  let error = $state('')
  let loading = $state(true)

  const monoColumns = ['name', 'rule', 'service', 'address']
  // Entrypoints have no protocol, so their items get an empty one.
  const protocolOf = (kind) => (kind.includes('/') ? kind.split('/')[0] : '')
  const protocols = $derived([...new Set(kinds.map(protocolOf))].filter(Boolean))
  const shownColumns = $derived(protocols.length > 1 ? ['protocol', ...columns] : columns)
  // Names repeat across protocols, so an item is identified by protocol and name.
  const keyOf = (i) => `${i.protocol}/${i.name}`
  const isSelected = (i) => selected !== null && keyOf(selected) === keyOf(i)

  async function load() {
    loading = true
    try {
      const lists = await Promise.all(
        kinds.map(async (kind) => {
          const list = await get(`/api/traefik/${kind}?per_page=1000`)
          return list.map((i) => ({ ...i, protocol: protocolOf(kind) }))
        }),
      )
      items = lists.flat()
      error = ''
    } catch (e) {
      if (e.unauthorized) return onerror(e)
      error = e.message
    } finally {
      loading = false
    }
  }
  $effect(() => {
    kinds
    selected = null
    load()
  })
  $effect(() => {
    protocol = initialProtocol
  })

  function cell(item, col) {
    if (col === 'protocol') return item.protocol.toUpperCase()
    if (col === 'servers') {
      const servers = item.loadBalancer?.servers ?? []
      const st = item.serverStatus ?? {}
      const addr = (s) => s.url ?? s.address
      const up = servers.filter((s) => st[addr(s)] === 'UP').length
      return `${servers.length}${Object.keys(st).length ? ` (${up} up)` : ''}`
    }
    const v = item[col]
    if (Array.isArray(v)) return v.join(', ')
    return v ?? ''
  }

  function statusClass(s) {
    return { enabled: 'good', warning: 'warn', disabled: 'bad' }[s] ?? 'muted'
  }

  const statuses = $derived([...new Set(items.map((i) => i.status).filter(Boolean))])
  const providers = $derived([...new Set(items.map((i) => i.provider).filter(Boolean))])
  // HTTP is one filter choice; TCP and UDP share another.
  const inProtocol = (i) => !protocol || (protocol === 'http' ? i.protocol === 'http' : i.protocol !== 'http')
  const shown = $derived(
    items.filter((i) => {
      if (!inProtocol(i)) return false
      if (status && i.status !== status) return false
      if (provider && i.provider !== provider) return false
      return !search || JSON.stringify(i).toLowerCase().includes(search.toLowerCase())
    }),
  )
</script>

<div class="toolbar">
  <input class="search" type="search" placeholder="Search…" bind:value={search} aria-label="Search" />
  {#if protocols.length > 1}
    <select bind:value={protocol} aria-label="Protocol">
      <option value="">All protocols</option>
      {#if protocols.includes('http')}<option value="http">HTTP</option>{/if}
      {#if protocols.some((p) => p !== 'http')}<option value="tcp-udp">TCP & UDP</option>{/if}
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
          <tr>{#each shownColumns as c}<th scope="col">{c}</th>{/each}</tr>
        </thead>
        <tbody>
          {#each shown as item (keyOf(item))}
            <tr class:sel={isSelected(item)} onclick={() => (selected = item)}>
              {#each shownColumns as c}
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
            <tr><td colspan={shownColumns.length} class="muted">Nothing matches.</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}

{#if selected}
  <section class="card detail" aria-label={`Details for ${selected.name}`}>
    <div class="detail-head">
      <h2 class="mono">{selected.name} <span class="muted">{selected.protocol.toUpperCase()}</span></h2>
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
