<script>
  import { status, post, get } from './api.js'
  import Login from './Login.svelte'
  import Dashboard from './Dashboard.svelte'
  import ObjectList from './ObjectList.svelte'
  import ConfigEditor from './ConfigEditor.svelte'
  import StaticConfig from './StaticConfig.svelte'
  import AuditLog from './AuditLog.svelte'
  import Settings from './Settings.svelte'

  const pages = [
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'objects', label: 'Traefik objects' },
    { id: 'config', label: 'Config files' },
    { id: 'static', label: 'Static config' },
    { id: 'audit', label: 'Audit log' },
    { id: 'settings', label: 'Settings' },
  ]

  let auth = $state(null)
  let error = $state('')
  let version = $state('')
  let page = $state(pageFromHash())

  function pageFromHash() {
    const [id, sub] = location.hash.replace(/^#\//, '').split('/')
    return { ...(pages.find((p) => p.id === id) ?? pages[0]), sub }
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

  $effect(() => {
    if (!auth?.authenticated) return
    get('/api/traefik/version')
      .then((v) => (version = v.Version ?? ''))
      .catch(() => (version = ''))
  })

  async function logout() {
    error = ''
    try {
      await post('/api/auth/logout')
    } catch (e) {
      error = `Could not sign out: ${e.message}`
    }
    await refresh()
  }
  refresh()
</script>

<div class="shell">
  <header class:bare={!auth?.authenticated}>
    <div class="brand">
      <svg width="26" height="26" viewBox="0 0 28 28" aria-hidden="true">
        <rect x="3" y="5" width="22" height="4" rx="2" fill="var(--accent)" />
        <rect x="3" y="12" width="16" height="4" rx="2" fill="var(--accent)" opacity="0.7" />
        <rect x="3" y="19" width="10" height="4" rx="2" fill="var(--accent)" opacity="0.45" />
      </svg>
      <span>Traefik WebUI</span>
    </div>
    {#if auth?.authenticated}
      <div class="right">
        {#if version}<span class="conn"><i></i>Connected to Traefik {version}</span>{/if}
        <button class="small" onclick={logout}>Sign out</button>
      </div>
    {/if}
  </header>
  {#if auth?.authenticated && error}
    <p class="bad notice" role="alert">{error}</p>
  {/if}

  {#if auth?.authenticated}
    <nav aria-label="Pages">
      {#each pages as p}
        <a href={`#/${p.id}`} aria-current={p.id === page.id ? 'page' : undefined}>{p.label}</a>
      {/each}
    </nav>
  {/if}

  {#if !auth}
    <main class="center"><p class="muted">{error || 'Loading…'}</p></main>
  {:else if !auth.authenticated}
    <main class="center"><Login setup={auth.setupRequired} onsuccess={refresh} /></main>
    <footer>
      <span>First visit? You will be asked to create the admin password (12 to 72 characters).</span>
    </footer>
  {:else}
    <main>
      {#if page.id === 'dashboard'}
        <Dashboard onerror={expired} />
      {:else if page.id === 'config'}
        <ConfigEditor onerror={expired} />
      {:else if page.id === 'static'}
        <StaticConfig onerror={expired} />
      {:else if page.id === 'audit'}
        <AuditLog onerror={expired} />
      {:else if page.id === 'settings'}
        <Settings onerror={expired} />
      {:else}
        <ObjectList type={page.sub} onerror={expired} />
      {/if}
    </main>
  {/if}
</div>

<style>
  .shell { min-height: 100vh; display: flex; flex-direction: column; }
  header { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; padding: 16px 32px; border-bottom: 1px solid var(--border); background: var(--surface); }
  .notice { margin: 0; padding: 10px 32px; font-size: 14px; }
  header.bare { background: transparent; border-bottom: 0; padding: 20px 32px; }
  .brand { display: flex; align-items: center; gap: 12px; font-weight: 600; font-size: 17px; }
  .right { display: flex; align-items: center; gap: 16px; }
  .conn { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--muted); }
  .conn i { width: 8px; height: 8px; border-radius: 50%; background: var(--good); }
  nav { display: flex; flex-wrap: wrap; gap: 4px; padding: 0 24px; border-bottom: 1px solid var(--border); background: var(--surface); }
  nav a { padding: 14px 12px; text-decoration: none; font-size: 14px; color: var(--muted); border-bottom: 2px solid transparent; }
  nav a:hover { color: var(--text); }
  nav a[aria-current] { color: var(--text); font-weight: 600; border-bottom-color: var(--accent); }
  main { flex: 1; width: 100%; max-width: 1280px; margin: 0 auto; padding: 32px; display: flex; flex-direction: column; gap: 24px; }
  main.center { align-items: center; justify-content: center; padding: 24px; }
  footer { padding: 16px 32px; font-size: 12px; color: var(--muted); display: flex; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
  @media (max-width: 600px) {
    main { padding: 20px 16px; }
    header, nav { padding-left: 16px; padding-right: 16px; }
  }
</style>
