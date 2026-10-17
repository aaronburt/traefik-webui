<script>
  import { get } from './api.js'

  let { onerror } = $props()
  let info = $state(null)
  let error = $state('')

  async function load() {
    try {
      info = await get('/api/static')
    } catch (e) {
      if (e.unauthorized) return onerror(e)
      error = e.message
    }
  }
  load()
</script>

<details class="card static">
  <summary>Static config (read-only)</summary>
  {#if error}
    <p class="bad" role="alert">{error}</p>
  {:else if !info}
    <p class="muted">Loading…</p>
  {:else if !info.configured}
    <p class="muted">
      No static config file is set. To show it, mount traefik.yml into the UI container read-only and set
      <code>STATIC_CONFIG</code> to its path. Traefik reads this file once at startup, so changes need a Traefik restart.
    </p>
  {:else if info.error}
    <p class="bad" role="alert">{info.error}</p>
  {:else}
    <p class="muted">
      Traefik reads this file once at startup, so changes here need a Traefik restart. Secret values are masked and
      comments are not shown.
    </p>
    <pre>{info.content}</pre>
  {/if}
</details>

<style>
  .static {
    margin-bottom: 20px;
  }
  summary {
    cursor: pointer;
    font-weight: 600;
  }
  p {
    margin: 12px 0 0;
    font-size: 14px;
  }
  code {
    font-family: 'IBM Plex Mono', ui-monospace, monospace;
    font-size: 12.5px;
  }
  pre {
    margin: 12px 0 0;
    padding: 14px;
    border-radius: 8px;
    background: var(--code);
    font-family: 'IBM Plex Mono', ui-monospace, monospace;
    font-size: 12px;
    overflow: auto;
  }
</style>
