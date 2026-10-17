<script>
  import { status, post } from './api.js'
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
    }
  }
  const expired = () => (auth = { ...auth, authenticated: false })

  async function logout() {
    await post('/api/auth/logout')
    await refresh()
  }
  refresh()
</script>

<main>
  <header>
    <h1>Traefik WebUI</h1>
    {#if auth?.authenticated}<button onclick={logout}>Sign out</button>{/if}
  </header>

  {#if !auth}
    <p>{error || 'Loading…'}</p>
  {:else if !auth.authenticated}
    <Login setup={auth.setupRequired} onsuccess={refresh} />
  {:else}
    <nav aria-label="Pages">
      {#each pages as p}
        <a href={`#/${p.id}`} aria-current={p.id === page.id ? 'page' : undefined}>{p.label}</a>
      {/each}
    </nav>
    <h2>{page.label}</h2>
    {#if page.id === 'dashboard'}
      <Dashboard onerror={expired} />
    {:else if page.id === 'config'}
      <ConfigEditor onerror={expired} />
    {:else}
      <ObjectList kind={page.kind} onerror={expired} />
    {/if}
  {/if}
</main>

<style>
  :global(:root) {
    --fg: #1b1f23; --bg: #fff; --border: #d0d7de; --hover: #f3f4f6; --muted: #6b7280;
    --good: #1a7f37; --warn: #9a6700; --bad: #b00020;
  }
  @media (prefers-color-scheme: dark) {
    :global(:root) {
      --fg: #e6e8eb; --bg: #14171a; --border: #30363d; --hover: #1f242a; --muted: #9aa4af;
      --good: #4cc26b; --warn: #e3b341; --bad: #ff7b72;
    }
  }
  :global(body) { margin: 0; color: var(--fg); background: var(--bg); font-family: system-ui, sans-serif; }
  :global(input, select, textarea, button) { font: inherit; color: inherit; background: var(--bg); border: 1px solid var(--border); border-radius: 6px; padding: 0.4rem 0.6rem; }
  :global(button) { cursor: pointer; }
  :global(button:disabled) { opacity: 0.5; cursor: not-allowed; }
  :global(a) { color: inherit; }
  main { max-width: 72rem; margin: 0 auto; padding: 1rem; }
  header { display: flex; justify-content: space-between; align-items: center; }
  nav { display: flex; flex-wrap: wrap; gap: 0.25rem; border-bottom: 1px solid var(--border); margin-bottom: 1rem; }
  nav a { padding: 0.5rem 0.75rem; text-decoration: none; border-bottom: 2px solid transparent; }
  nav a[aria-current] { border-bottom-color: var(--fg); font-weight: 600; }
</style>
