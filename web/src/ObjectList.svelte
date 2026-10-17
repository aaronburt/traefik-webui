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

<div class="bar">
  <input type="search" placeholder="Search…" bind:value={search} aria-label="Search" />
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
  <span>{shown.length} of {items.length}</span>
</div>

{#if error}
  <p class="error" role="alert">Cannot load: {error}</p>
{:else if loading && !items.length}
  <p>Loading…</p>
{:else}
  <div class="scroll">
    <table>
      <thead><tr>{#each columns[kind] as c}<th>{c}</th>{/each}</tr></thead>
      <tbody>
        {#each shown as item (item.name)}
          <tr class:sel={selected?.name === item.name} onclick={() => (selected = item)}>
            {#each columns[kind] as c}
              <td class={c === 'status' ? item.status : ''}>{cell(item, c)}</td>
            {/each}
          </tr>
        {:else}
          <tr><td colspan={columns[kind].length}>Nothing matches.</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

{#if selected}
  <aside>
    <button class="close" onclick={() => (selected = null)}>Close</button>
    <h2>{selected.name}</h2>
    {#if selected.error?.length}
      <p class="error" role="alert">{selected.error.join('; ')}</p>
    {/if}
    <pre>{JSON.stringify(selected, null, 2)}</pre>
  </aside>
{/if}

<style>
  .bar { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; margin-bottom: 1rem; }
  .scroll { overflow-x: auto; }
  table { border-collapse: collapse; width: 100%; }
  th, td { text-align: left; padding: 0.4rem 0.6rem; border-bottom: 1px solid var(--border); white-space: nowrap; }
  td:nth-child(3), td:nth-child(4) { white-space: normal; word-break: break-word; }
  tbody tr { cursor: pointer; }
  tbody tr:hover, tr.sel { background: var(--hover); }
  .enabled { color: var(--good); }
  .warning { color: var(--warn); }
  .disabled { color: var(--bad); }
  .error { color: var(--bad); }
  aside { margin-top: 1rem; border: 1px solid var(--border); border-radius: 8px; padding: 0 1rem 1rem; position: relative; }
  .close { position: absolute; top: 0.75rem; right: 0.75rem; }
  pre { overflow: auto; background: var(--hover); padding: 0.75rem; border-radius: 6px; }
</style>
