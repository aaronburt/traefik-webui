<script>
  import { status, get, post } from './api.js'
  import Login from './Login.svelte'

  let auth = $state(null) // { setupRequired, authenticated }
  let version = $state(null)
  let error = $state(null)

  async function refresh() {
    try {
      auth = await status()
      if (auth.authenticated) await loadVersion()
    } catch (e) {
      error = e.message
    }
  }

  async function loadVersion() {
    try {
      version = (await get('/api/traefik/version')).Version
      error = null
    } catch (e) {
      if (e.unauthorized) auth = { ...auth, authenticated: false }
      else error = e.message
    }
  }

  async function logout() {
    await post('/api/auth/logout')
    version = null
    await refresh()
  }

  refresh()
</script>

<main>
  <h1>Traefik WebUI</h1>
  {#if !auth}
    <p>{error ?? 'Loading…'}</p>
  {:else if !auth.authenticated}
    <Login setup={auth.setupRequired} onsuccess={refresh} />
  {:else}
    <p>
      {#if version}Connected to Traefik {version}{:else if error}Cannot reach Traefik: {error}{:else}Connecting…{/if}
    </p>
    <button onclick={logout}>Sign out</button>
  {/if}
</main>

<style>
  main { font-family: system-ui, sans-serif; max-width: 40rem; margin: 3rem auto; padding: 0 1rem; }
</style>
