<script>
  let version = $state(null)
  let error = $state(null)

  async function load() {
    try {
      const res = await fetch('/api/traefik/version')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      version = (await res.json()).Version
    } catch (e) {
      error = e.message
    }
  }
  load()
</script>

<main>
  <h1>Traefik WebUI</h1>
  {#if version}
    <p>Connected to Traefik {version}</p>
  {:else if error}
    <p>Cannot reach Traefik: {error}</p>
  {:else}
    <p>Connecting…</p>
  {/if}
</main>

<style>
  main { font-family: system-ui, sans-serif; max-width: 40rem; margin: 3rem auto; padding: 0 1rem; }
</style>
