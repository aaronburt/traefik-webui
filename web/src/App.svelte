<script>
  import { status, post, get } from './api.js'
  import Login from './Login.svelte'
  import Dashboard from './Dashboard.svelte'
  import ObjectList from './ObjectList.svelte'
  import ConfigEditor from './ConfigEditor.svelte'
  import StaticConfig from './StaticConfig.svelte'

  // The Objects tab is read-only and shows one kind at a time. Routers and services list every protocol.
  const objectKinds = {
    routers: {
      label: 'Routers',
      kinds: ['http/routers', 'tcp/routers', 'udp/routers'],
      columns: ['name', 'status', 'rule', 'service', 'provider'],
    },
    services: {
      label: 'Services',
      kinds: ['http/services', 'tcp/services', 'udp/services'],
      columns: ['name', 'status', 'type', 'servers', 'provider'],
    },
    middlewares: { label: 'Middlewares', kinds: ['http/middlewares'], columns: ['name', 'status', 'type', 'provider'] },
    entrypoints: { label: 'Entrypoints', kinds: ['entrypoints'], columns: ['name', 'address'] },
  }

  const pages = [
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'objects', label: 'Objects' },
    { id: 'config', label: 'Config files' },
  ]

  let auth = $state(null) // { setupRequired, authenticated }
  let error = $state('')
  let version = $state('')
  let page = $state(pageFromHash())
  let kind = $state(kindFromHash())
  let protocol = $state(protocolFromHash())

  // Hashes look like #/objects?kind=routers&protocol=tcp-udp.
  function hashParts() {
    const [path, query = ''] = location.hash.replace(/^#\//, '').split('?')
    return { id: path, query: new URLSearchParams(query) }
  }
  function pageFromHash() {
    return pages.find((p) => p.id === hashParts().id) ?? pages[0]
  }
  function kindFromHash() {
    const k = hashParts().query.get('kind')
    return Object.hasOwn(objectKinds, k) ? k : 'routers'
  }
  function protocolFromHash() {
    return hashParts().query.get('protocol') ?? ''
  }
  $effect(() => {
    const on = () => {
      page = pageFromHash()
      kind = kindFromHash()
      protocol = protocolFromHash()
    }
    addEventListener('hashchange', on)
    return () => removeEventListener('hashchange', on)
  })

  async function refresh() {
    try {
      auth = await status()
    } catch (e) {
      error = e.message
      return
    }
    if (auth.authenticated) loadVersion()
  }

  async function loadVersion() {
    try {
      version = (await get('/api/traefik/version')).Version
    } catch (e) {
      if (e.unauthorized) auth = { ...auth, authenticated: false }
    }
  }

  const expired = () => (auth = { ...auth, authenticated: false })

  async function logout() {
    await post('/api/auth/logout')
    version = ''
    await refresh()
  }
  refresh()
</script>

<header class="top">
  <div class="brand">
    <svg width="26" height="26" viewBox="0 0 28 28" aria-hidden="true">
      <rect x="3" y="5" width="22" height="4" rx="2" fill="var(--accent)" />
      <rect x="3" y="12" width="16" height="4" rx="2" fill="var(--accent)" opacity="0.7" />
      <rect x="3" y="19" width="10" height="4" rx="2" fill="var(--accent)" opacity="0.45" />
    </svg>
    <span>Traefik WebUI</span>
  </div>
  {#if auth?.authenticated}
    <div class="top-actions">
      {#if version}
        <span class="pill"><span class="dot good"></span>Connected to Traefik {version}</span>
      {/if}
      <button onclick={logout}>Sign out</button>
    </div>
  {/if}
</header>

{#if !auth}
  <main><p class="muted">{error || 'Loading…'}</p></main>
{:else if !auth.authenticated}
  <main>
    <Login setup={auth.setupRequired} onsuccess={refresh} />
  </main>
{:else}
  <nav class="tabs" aria-label="Pages">
    {#each pages as p}
      <a href={`#/${p.id}`} aria-current={p.id === page.id ? 'page' : undefined}>{p.label}</a>
    {/each}
  </nav>
  <main>
    <h1>{page.label}</h1>
    {#if page.id === 'dashboard'}
      <Dashboard onerror={expired} />
    {:else if page.id === 'config'}
      <StaticConfig onerror={expired} />
      <ConfigEditor onerror={expired} />
    {:else}
      <nav class="kinds" aria-label="Object kind">
        {#each Object.entries(objectKinds) as [id, k]}
          <a href={`#/objects?kind=${id}`} aria-current={id === kind ? 'page' : undefined}>{k.label}</a>
        {/each}
      </nav>
      <ObjectList kinds={objectKinds[kind].kinds} columns={objectKinds[kind].columns} initialProtocol={protocol} onerror={expired} />
    {/if}
  </main>
{/if}

<style>
  .top {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    padding: 16px 32px;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 17px;
    font-weight: 600;
  }
  .top-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 16px;
  }
  .tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    padding: 0 24px;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
  }
  .tabs a {
    padding: 14px 12px;
    border-bottom: 2px solid transparent;
    color: var(--muted);
    font-size: 14px;
    text-decoration: none;
  }
  .tabs a[aria-current] {
    border-bottom-color: var(--accent);
    color: var(--text);
    font-weight: 600;
  }
  .kinds {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin: 0 0 20px;
  }
  .kinds a {
    padding: 6px 14px;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    color: var(--muted);
    font-size: 14px;
    text-decoration: none;
  }
  .kinds a[aria-current] {
    border-color: var(--accent);
    color: var(--text);
    font-weight: 600;
  }
  main {
    max-width: 1216px;
    margin: 0 auto;
    padding: 32px;
  }
  @media (max-width: 700px) {
    .top,
    main {
      padding: 16px;
    }
  }
</style>
