<script>
  import { get, send } from './api.js'

  let { onerror } = $props()

  const base = '/api/config/files'
  let files = $state([])
  let name = $state('') // selected (or new) file name
  let isNew = $state(false)
  let text = $state('')
  let original = $state('')
  let problems = $state([])
  let diff = $state(null) // array of {op,text} while reviewing
  let result = $state(null) // outcome of the last save/restore
  let backups = $state([])
  let message = $state('')
  let busy = $state(false)
  let newName = $state('')

  const dirty = $derived(text !== original)
  const url = (suffix = '') => `${base}/${encodeURIComponent(name)}${suffix}`

  async function call(method, path, body) {
    try {
      return await send(method, path, body)
    } catch (e) {
      if (e.unauthorized) onerror(e)
      throw e
    }
  }

  async function loadFiles() {
    files = await get(base).catch((e) => (e.unauthorized ? onerror(e) : ((message = e.message), [])))
  }
  loadFiles()

  async function open(f, fresh = false) {
    if (dirty && !confirm('Discard unsaved changes?')) return
    name = f
    isNew = fresh
    diff = result = null
    problems = []
    message = ''
    if (fresh) {
      text = original = ''
      backups = []
      return
    }
    const r = await call('GET', url())
    text = original = r.data?.content ?? ''
    await loadBackups()
    validate()
  }

  async function loadBackups() {
    const r = await call('GET', url('/backups'))
    backups = r.ok ? r.data : []
  }

  let timer
  function validate() {
    clearTimeout(timer)
    timer = setTimeout(async () => {
      if (!name) return
      const r = await call('POST', url('/validate'), { content: text })
      problems = r.data?.problems ?? []
    }, 300)
  }
  $effect(() => {
    text
    validate()
    return () => clearTimeout(timer)
  })

  async function review() {
    busy = true
    result = null
    const r = await call('POST', url('/diff'), { content: text })
    diff = r.data?.lines ?? []
    busy = false
  }

  async function apply() {
    busy = true
    const r = await call('PUT', url(), { content: text })
    busy = false
    if (r.ok) {
      result = r.data
      original = text
      diff = null
      isNew = false
      await Promise.all([loadFiles(), loadBackups()])
    } else if (r.status === 422) {
      problems = r.data.problems
      diff = null
      message = 'Not saved: fix the problems below.'
    } else {
      message = r.text.trim()
    }
  }

  async function restore(id) {
    if (!confirm('Restore this version? The current one is kept as a backup.')) return
    const r = await call('POST', url(`/backups/${id}/restore`))
    if (r.ok) {
      result = r.data
      text = original = (await call('GET', url())).data.content
      diff = null
      await loadBackups()
    } else message = r.text.trim()
  }

  async function viewBackup(id) {
    const r = await call('GET', url(`/backups/${id}`))
    if (r.ok) {
      text = r.data.content
      diff = null
      message = 'Loaded an older version into the editor. Review and save to apply it.'
    }
  }

  async function remove() {
    if (!confirm(`Delete ${name}? A backup is kept.`)) return
    const r = await call('DELETE', url())
    if (r.ok) {
      name = text = original = ''
      diff = result = null
      backups = []
      await loadFiles()
    } else message = r.text.trim()
  }

  function startNew(e) {
    e.preventDefault()
    const n = newName.trim()
    if (!n) return
    open(/\.ya?ml$/.test(n) ? n : `${n}.yml`, true)
    newName = ''
  }

  // Guided form: generates a complete route file (router, service, optional
  // redirect and TLS) so no existing YAML needs merging.
  let route = $state({ id: '', host: '', backend: '', entrypoint: 'websecure', resolver: '', redirect: true })
  const q = JSON.stringify
  function routeYaml(r) {
    const lines = ['http:', '  routers:']
    const rule = q(`Host(\`${r.host}\`)`)
    lines.push(`    ${r.id}:`, `      rule: ${rule}`, `      service: ${r.id}`, `      entryPoints: [${q(r.entrypoint)}]`)
    if (r.resolver) lines.push('      tls:', `        certResolver: ${q(r.resolver)}`)
    if (r.redirect && r.entrypoint !== 'web') {
      lines.push(`    ${r.id}-http:`, `      rule: ${rule}`, '      entryPoints: ["web"]', `      middlewares: [${q(r.id + '-https')}]`, `      service: ${r.id}`)
    }
    lines.push('  services:', `    ${r.id}:`, '      loadBalancer:', '        servers:', `          - url: ${q(r.backend)}`)
    if (r.redirect && r.entrypoint !== 'web') {
      lines.push('  middlewares:', `    ${r.id}-https:`, '      redirectScheme:', '        scheme: https', '        permanent: true')
    }
    return lines.join('\n') + '\n'
  }
  const routeOk = $derived(/^[a-z0-9][a-z0-9-]*$/.test(route.id) && route.host && /^https?:\/\/.+/.test(route.backend))
  async function createRoute(e) {
    e.preventDefault()
    await open(`${route.id}.yml`, true)
    text = routeYaml(route)
  }
</script>

<div class="layout">
  <nav class="card" aria-label="Config files">
    <h2>Files</h2>
    <ul>
      {#each files as f}
        <li><button class:active={f.name === name} onclick={() => open(f.name)}>{f.name}</button></li>
      {:else}
        <li class="muted">No files yet.</li>
      {/each}
    </ul>
    <form onsubmit={startNew}>
      <input placeholder="new-file.yml" bind:value={newName} aria-label="New file name" />
      <button>New</button>
    </form>

    <details>
      <summary>Quick route</summary>
      <form onsubmit={createRoute} class="stack">
        <label>Name <input bind:value={route.id} placeholder="myapp" /></label>
        <label>Domain <input bind:value={route.host} placeholder="app.example.com" /></label>
        <label>Backend URL <input bind:value={route.backend} placeholder="http://10.0.0.5:8080" /></label>
        <label>Entrypoint <input bind:value={route.entrypoint} /></label>
        <label>Cert resolver (optional) <input bind:value={route.resolver} placeholder="letsencrypt" /></label>
        <label class="row"><input type="checkbox" bind:checked={route.redirect} /> Redirect HTTP to HTTPS</label>
        <button disabled={!routeOk}>Create route file</button>
      </form>
    </details>
  </nav>

  <section class="card">
    {#if !name}
      <p>Select a file, or create a new one. Traefik reloads changes to these files automatically.</p>
    {:else}
      <h2 class="file-name">{name}{#if isNew} <small>(new)</small>{/if}{#if dirty} <small>• unsaved</small>{/if}</h2>
      <textarea bind:value={text} spellcheck="false" aria-label="YAML content" rows="22"></textarea>

      {#if problems.length}
        <ul class="problems" role="alert">
          {#each problems as p}<li><code>{p.path || 'file'}</code> {p.message}</li>{/each}
        </ul>
      {:else if dirty}
        <p class="good">No problems found.</p>
      {/if}
      {#if message}<p>{message}</p>{/if}

      <div class="actions">
        <button class="primary" disabled={!dirty || problems.length > 0 || busy} onclick={review}>Review changes</button>
        {#if !isNew}<button class="danger" onclick={remove}>Delete file</button>{/if}
      </div>

      {#if diff}
        <h3>Changes</h3>
        <pre class="diff">{#each diff as l}<span class={l.op === '+' ? 'add' : l.op === '-' ? 'del' : ''}>{l.op} {l.text}
</span>{/each}</pre>
        <button class="primary" disabled={busy} onclick={apply}>Confirm and apply</button>
        <button onclick={() => (diff = null)}>Cancel</button>
      {/if}

      {#if result?.applied}
        <div class="outcome" role="status">
          <p class={result.warnings?.length ? 'warn' : 'good'}>
            {result.warnings?.length ? 'Saved, but Traefik reports problems:' : 'Saved. Traefik accepted the change.'}
          </p>
          {#if result.warnings?.length}
            <ul>{#each result.warnings as w}<li>{w}</li>{/each}</ul>
            {#if backups.length}<button onclick={() => restore(backups[0].id)}>Revert to previous version</button>{/if}
          {/if}
        </div>
      {/if}

      {#if backups.length}
        <details>
          <summary>History ({backups.length})</summary>
          <ul class="history">
            {#each backups as b}
              <li>
                <span class="when">{new Date(b.time).toLocaleString()} · {b.size} bytes</span>
                <button onclick={() => viewBackup(b.id)}>View</button>
                <button onclick={() => restore(b.id)}>Restore</button>
              </li>
            {/each}
          </ul>
        </details>
      {/if}
    {/if}
  </section>
</div>

<style>
  .layout {
    display: grid;
    grid-template-columns: 220px 1fr;
    gap: 20px;
    align-items: start;
  }
  @media (max-width: 700px) {
    .layout {
      grid-template-columns: 1fr;
    }
  }
  nav,
  section {
    min-width: 0;
  }
  h2 {
    margin: 0 0 12px;
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--muted);
  }
  .file-name {
    margin: 0 0 12px;
    font-family: 'IBM Plex Mono', ui-monospace, monospace;
    font-size: 16px;
    font-weight: 600;
    letter-spacing: 0;
    text-transform: none;
    color: var(--text);
    overflow-wrap: anywhere;
  }
  h3 {
    margin: 24px 0 10px;
    font-size: 15px;
    font-weight: 600;
  }
  p {
    margin: 0 0 12px;
  }
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  code {
    font-family: 'IBM Plex Mono', ui-monospace, monospace;
    font-size: 12.5px;
  }
  nav ul {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  nav li button {
    width: 100%;
    height: 36px;
    padding: 0 10px;
    border-color: transparent;
    background: none;
    font-family: 'IBM Plex Mono', ui-monospace, monospace;
    font-size: 13px;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  nav li button:hover {
    background: var(--hover);
  }
  nav li button.active {
    background: var(--hover);
    box-shadow: inset 3px 0 0 var(--accent);
    font-weight: 600;
  }
  nav form:not(.stack) {
    display: flex;
    gap: 8px;
    margin-top: 12px;
  }
  nav form:not(.stack) input {
    flex: 1;
    min-width: 0;
    height: 36px;
    font-size: 13px;
  }
  nav form:not(.stack) button {
    height: 36px;
  }
  details {
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--border);
  }
  summary {
    margin-bottom: 12px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
  }
  .stack {
    display: grid;
    gap: 12px;
  }
  .stack input {
    width: 100%;
  }
  label {
    display: grid;
    gap: 4px;
    font-size: 13px;
    font-weight: 500;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  textarea {
    width: 100%;
    min-height: 420px;
    padding: 14px;
    border-radius: 12px;
    background: var(--code);
    font-family: 'IBM Plex Mono', ui-monospace, monospace;
    font-size: 13px;
    line-height: 1.6;
  }
  .problems {
    padding: 12px 16px;
    border: 1px solid var(--bad);
    border-radius: 8px;
    color: var(--bad);
    font-size: 14px;
  }
  .problems li + li {
    margin-top: 4px;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin: 12px 0;
  }
  .danger {
    color: var(--bad);
  }
  .diff {
    margin: 0 0 12px;
    padding: 8px 0;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--code);
    font-family: 'IBM Plex Mono', ui-monospace, monospace;
    font-size: 12.5px;
    line-height: 1.6;
    overflow-x: auto;
  }
  .diff span {
    display: block;
    padding: 0 14px;
    white-space: pre;
  }
  .add {
    background: color-mix(in srgb, var(--good) 18%, transparent);
  }
  .del {
    background: color-mix(in srgb, var(--bad) 18%, transparent);
  }
  .outcome {
    margin-top: 16px;
    padding: 14px 18px;
    border: 1px solid var(--border);
    border-radius: 12px;
  }
  .history li {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    padding: 8px 0;
    border-top: 1px solid var(--border);
  }
  .when {
    flex: 1;
    font-size: 14px;
  }
  .history button {
    height: 32px;
    padding: 0 12px;
    font-size: 13px;
  }
  small {
    font-weight: normal;
    color: var(--muted);
  }
</style>
