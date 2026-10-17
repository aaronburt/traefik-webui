<script>
  import { status, post, get } from './api.js'
  import Login from './Login.svelte'
  import Dashboard from './Dashboard.svelte'
  import ObjectList from './ObjectList.svelte'
  import ConfigEditor from './ConfigEditor.svelte'

  const pages = [
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'routers', label: 'Routers', kind: 'http/routers' },
    { id: 'services', label: 'Services', kind: 'http/services' },
    { id: 'middlewares', label: 'Middlewares', kind: 'http/middlewares' },
    { id: 'tcp-routers', label: 'TCP routers', kind: 'tcp/routers' },
    { id: 'tcp-services', label: 'TCP services', kind: 'tcp/services' },
    { id: 'udp-routers', label: 'UDP routers', kind: 'udp/routers' },
    { id: 'udp-services', label: 'UDP services', kind: 'udp/services' },
    { id: 'entrypoints', label: 'Entrypoints', kind: 'entrypoints' },
    { id: 'config', label: 'Config files' },
  ]

  let auth = $state(null) // { setupRequired, authenticated }
  let error = $state('')
  let version = $state('')
  let page = $state(pageFromHash())

  function pageFromHash() {
    const id = location.hash.replace(/^#\//, '')
    return pages.find((p) => p.id === id) ?? pages[0]
  }
  $effect(() => {
    const on = () => (page = pageFromHash())
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
      <ConfigEditor onerror={expired} />
    {:else}
      <ObjectList kind={page.kind} onerror={expired} />
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
