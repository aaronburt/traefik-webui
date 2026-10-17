<script>
  import { get, send } from './api.js'

  let { onerror } = $props()

  const base = '/api/config/files'
  let files = $state([])
  let name = $state('')
  let isNew = $state(false)
  let text = $state('')
  let original = $state('')
  let problems = $state([])
  let diff = $state(null)
  let result = $state(null)
  let backups = $state([])
  let message = $state('')
  let busy = $state(false)
  let newName = $state('')
  let quick = $state(false)

  const dirty = $derived(text !== original)
  const url = (suffix = '') => `${base}/${encodeURIComponent(name)}${suffix}`
  const added = $derived(diff?.filter((l) => l.op === '+').length ?? 0)
  const removed = $derived(diff?.filter((l) => l.op === '-').length ?? 0)

  async function call(method, path, body) {
    try {
      return await send(method, path, body)
    } catch (e) {
      if (e.unauthorized) onerror(e)
      throw e
    }
  }

  // Event handlers run unawaited, so a failed request must end up on screen, not as an unhandled rejection.
  // A 401 has already been handed to onerror by call().
  const guard =
    (fn) =>
    async (...args) => {
      try {
        return await fn(...args)
      } catch (e) {
        if (!e.unauthorized) message = `Request failed: ${e.message}`
      }
    }

  async function loadFiles() {
    try {
      files = await get(base)
    } catch (e) {
      if (e.unauthorized) onerror(e)
      else message = e.message
    }
  }
  loadFiles()

  // Opens a file in the editor. Returns false when the user keeps their unsaved changes instead.
  const open = guard(async (f, fresh = false) => {
    if (dirty && !confirm('Discard unsaved changes?')) return false
    quick = false
    name = f
    isNew = fresh
    diff = result = null
    problems = []
    message = ''
    if (fresh) {
      text = original = ''
      backups = []
      return true
    }
    const r = await call('GET', url())
    text = original = r.data?.content ?? ''
    await loadBackups()
    validate()
    return true
  })

  async function loadBackups() {
    const r = await call('GET', url('/backups'))
    backups = r.ok ? r.data : []
  }

  // Validation runs 300 ms after the last keystroke. A response only counts if nothing newer
  // was requested meanwhile and the same file is still open.
  let timer
  let seq = 0
  function validate() {
    clearTimeout(timer)
    const mine = ++seq
    timer = setTimeout(
      guard(async () => {
        if (!name) return
        const forName = name
        const r = await call('POST', url('/validate'), { content: text })
        if (mine === seq && forName === name) problems = r.data?.problems ?? []
      }),
      300,
    )
  }
  $effect(() => {
    text // re-validate whenever the text changes
    validate()
    return () => clearTimeout(timer)
  })

  // Warn before unsaved edits are lost by reloading, closing the tab or following a link to another page.
  $effect(() => {
    if (!dirty) return
    const beforeUnload = (e) => e.preventDefault()
    const onClick = (e) => {
      const href = e.target.closest?.('a[href^="#/"]')?.getAttribute('href')
      if (href && !href.startsWith('#/config') && !confirm('Discard unsaved changes?')) e.preventDefault()
    }
    addEventListener('beforeunload', beforeUnload)
    document.addEventListener('click', onClick, true)
    return () => {
      removeEventListener('beforeunload', beforeUnload)
      document.removeEventListener('click', onClick, true)
    }
  })

  // Typing invalidates a shown diff and the last save result: Apply sends the current text,
  // so the diff on screen must always describe it.
  function edited() {
    diff = null
    result = null
  }

  const review = guard(async () => {
    busy = true
    result = null
    try {
      const r = await call('POST', url('/diff'), { content: text })
      diff = r.data?.lines ?? []
    } finally {
      busy = false
    }
  })

  const apply = guard(async () => {
    busy = true
    try {
      const r = await call('PUT', url(), { content: text })
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
    } finally {
      busy = false
    }
  })

  const restore = guard(async (id) => {
    const warning = dirty ? 'Restore this version? Your unsaved edits will be lost, and the saved version is kept as a backup.' : 'Restore this version? The current one is kept as a backup.'
    if (!confirm(warning)) return
    const r = await call('POST', url(`/backups/${id}/restore`))
    if (r.ok) {
      result = r.data
      text = original = (await call('GET', url())).data.content
      diff = null
      await loadBackups()
    } else message = r.text.trim()
  })

  const viewBackup = guard(async (id) => {
    const r = await call('GET', url(`/backups/${id}`))
    if (r.ok) {
      text = r.data.content
      diff = result = null
      message = 'Loaded an older version into the editor. Review and save to apply it.'
    }
  })

  const remove = guard(async () => {
    if (!confirm(`Delete ${name}? A backup is kept.`)) return
    const r = await call('DELETE', url())
    if (r.ok) {
      name = text = original = ''
      diff = result = null
      backups = []
      await loadFiles()
    } else message = r.text.trim()
  })

  // Typing the name of an existing file opens it rather than starting an empty "new" file over it.
  async function startNew(e) {
    e.preventDefault()
    const n = newName.trim()
    if (!n) return
    const file = /\.ya?ml$/.test(n) ? n : `${n}.yml`
    if (await open(file, !files.some((f) => f.name === file))) newName = ''
  }

  let route = $state({ id: '', host: '', backend: '', entrypoint: 'websecure', resolver: '', redirect: true })
  let routeText = $state('')
  let routeEdited = $state(false)
  const q = JSON.stringify
  function routeYaml(r) {
    const id = r.id || 'myapp'
    const host = r.host || 'app.example.com'
    const backend = r.backend || 'http://10.0.0.5:8080'
    const lines = ['http:', '  routers:']
    const rule = q(`Host(\`${host}\`)`)
    lines.push(`    ${id}:`, `      rule: ${rule}`, `      service: ${id}`, `      entryPoints: [${q(r.entrypoint)}]`)
    if (r.resolver) lines.push('      tls:', `        certResolver: ${q(r.resolver)}`)
    if (r.redirect && r.entrypoint !== 'web') {
      lines.push(`    ${id}-http:`, `      rule: ${rule}`, '      entryPoints: ["web"]', `      middlewares: [${q(id + '-https')}]`, `      service: ${id}`)
    }
    lines.push('  services:', `    ${id}:`, '      loadBalancer:', '        servers:', `          - url: ${q(backend)}`)
    if (r.redirect && r.entrypoint !== 'web') {
      lines.push('  middlewares:', `    ${id}-https:`, '      redirectScheme:', '        scheme: https', '        permanent: true')
    }
    return lines.join('\n') + '\n'
  }
  $effect(() => {
    const y = routeYaml(route)
    if (!routeEdited) routeText = y
  })
  const routeOk = $derived(/^[a-z0-9][a-z0-9-]*$/.test(route.id) && route.host && route.entrypoint.trim() && /^https?:\/\/.+/.test(route.backend))
  async function createRoute(e) {
    e.preventDefault()
    const file = `${route.id}.yml`
    const exists = files.some((f) => f.name === file)
    if (exists && !confirm(`${file} already exists. Replace its content? You review a diff first and a backup is kept.`)) return
    // Stop if the user chose to keep their unsaved edits, so the route text does not land in the wrong file.
    if (!(await open(file, !exists))) return
    text = routeText
    routeEdited = false
  }
</script>

<div class="layout">
  <nav aria-label="Config files">
    <h1>Config files</h1>
    <div class="stack list">
      {#each files as f}
        <button class="file mono" class:active={!quick && f.name === name} onclick={() => open(f.name)}>
          {f.name}{#if !quick && f.name === name && dirty} <span class="warn">●</span>{/if}
        </button>
      {:else}
        <span class="muted none">No files yet.</span>
      {/each}
    </div>
    <form class="new" onsubmit={startNew}>
      <input placeholder="new-file.yml" bind:value={newName} aria-label="New file name" />
      <button>New</button>
    </form>
    <button class="link" class:on={quick} aria-pressed={quick} onclick={() => (quick = !quick)}>Quick route form</button>
  </nav>

  {#if quick}
    <section class="content">
      <div class="stack head">
        <h2>Quick route</h2>
        <p class="muted">Publish a service at a domain. This creates one config file with a router, a service and an optional HTTPS redirect.</p>
      </div>
      <div class="cols">
        <form class="card form" onsubmit={createRoute}>
          <label class="field">
            <span>Name</span>
            <input class="mono" bind:value={route.id} placeholder="myapp" />
            <small class="hint">Lowercase letters, digits and dashes. Becomes {route.id || 'myapp'}.yml.</small>
          </label>
          <label class="field"><span>Domain</span><input bind:value={route.host} placeholder="app.example.com" /></label>
          <label class="field"><span>Backend URL</span><input class="mono" bind:value={route.backend} placeholder="http://10.0.0.5:8080" /></label>
          <div class="two">
            <label class="field"><span>Entrypoint</span><input class="mono" bind:value={route.entrypoint} /></label>
            <label class="field"><span>Cert resolver <em class="muted">(optional)</em></span><input class="mono" bind:value={route.resolver} placeholder="letsencrypt" /></label>
          </div>
          <label class="check"><input type="checkbox" bind:checked={route.redirect} /> Redirect HTTP to HTTPS</label>
          <button class="primary" disabled={!routeOk}>Create route file</button>
          <span class="hint">You review a diff before anything is written.</span>
        </form>

        <div class="stack preview">
          <div class="row-between">
            <h3>Preview: <span class="mono">{route.id || 'myapp'}.yml</span></h3>
            {#if routeEdited}<button class="small" onclick={() => (routeEdited = false)}>Reset to form</button>{/if}
          </div>
          <textarea class="code" bind:value={routeText} oninput={() => (routeEdited = true)} spellcheck="false" aria-label="Route file YAML" rows="24"></textarea>
          <p class="muted help">The preview is editable. The file lands in the folder Traefik watches, so it goes live as soon as you apply it. A backup of any replaced file is kept.</p>
        </div>
      </div>
    </section>
  {:else}
    <section class="content">
      {#if !name}
        <p class="muted">Select a file, or create a new one. Traefik reloads changes to these files automatically.</p>
        {#if message}<p class="bad" role="alert">{message}</p>{/if}
      {:else}
        <div class="title">
          <h2 class="mono">{name}</h2>
          {#if isNew}<span class="muted note">new file</span>{/if}
          {#if dirty}<span class="warn note">● unsaved changes</span>{/if}
        </div>

        <textarea class="code" bind:value={text} oninput={edited} spellcheck="false" aria-label="YAML content" rows="22"></textarea>

        {#if problems.length}
          <ul class="problems" role="alert">
            {#each problems as p}<li><code>{p.path || 'file'}</code> {p.message}</li>{/each}
          </ul>
        {:else if dirty}
          <p class="good ok">
            <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" stroke-width="1.5" /><path d="M5 8.2l2 2 4-4.4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
            No problems found.
          </p>
        {/if}
        {#if message}<p class="muted">{message}</p>{/if}

        <div class="actions">
          <button disabled={!dirty || problems.length > 0 || busy} onclick={review}>Review changes</button>
          {#if !isNew}<button class="danger" onclick={remove}>Delete file</button>{/if}
        </div>

        {#if diff}
          <section class="card review" aria-label="Review">
            <div class="review-head">
              <h3>Changes to apply</h3>
              <span class="muted note"><span class="good">+{added}</span> · <span class="bad">−{removed}</span> · a backup is kept</span>
            </div>
            <div class="diff">
              {#each diff as l}
                <div class="line" class:add={l.op === '+'} class:del={l.op === '-'}>{l.op} {l.text}</div>
              {/each}
            </div>
            <div class="review-foot">
              <button class="primary" disabled={busy} onclick={apply}>Confirm and apply</button>
              <button onclick={() => (diff = null)}>Cancel</button>
            </div>
          </section>
        {/if}

        {#if result?.applied}
          <div class="banner" class:warn={result.warnings?.length} role="status">
            {#if result.warnings?.length}
              <svg width="20" height="20" viewBox="0 0 16 16" aria-hidden="true"><path d="M8 1.8l6.5 11.4h-13z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" /><path d="M8 6.4v3M8 11.2v.3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" /></svg>
            {:else}
              <svg class="good" width="20" height="20" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" stroke-width="1.5" /><path d="M5 8.2l2 2 4-4.4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
            {/if}
            <div class="stack msg">
              <strong>{result.warnings?.length ? 'Saved, but Traefik reports a problem' : 'Saved. Traefik accepted the change.'}</strong>
              {#each result.warnings ?? [] as w}<span class="muted">{w}</span>{/each}
              {#if result.warnings?.length && backups.length}
                <div><button class="small" onclick={() => restore(backups[0].id)}>Revert to previous version</button></div>
              {/if}
            </div>
          </div>
        {/if}

        {#if backups.length}
          <details class="card history">
            <summary>History ({backups.length})</summary>
            {#each backups as b}
              <div class="hrow">
                <span>{new Date(b.time).toLocaleString()} · {b.size} bytes</span>
                <span class="btns">
                  <button class="small" onclick={() => viewBackup(b.id)}>View</button>
                  <button class="small" onclick={() => restore(b.id)}>Restore</button>
                </span>
              </div>
            {/each}
          </details>
        {/if}
      {/if}
    </section>
  {/if}
</div>

<style>
  .layout { display: flex; flex-wrap: wrap; gap: 24px; align-items: flex-start; }
  nav { flex: 0 0 220px; display: flex; flex-direction: column; gap: 16px; }
  nav h1 { font-size: 20px; font-weight: 600; }
  .list { gap: 4px; }
  .file { text-align: left; height: 40px; padding: 0 12px; border-radius: 8px; border: 1px solid transparent; background: transparent; font-size: 13px; font-weight: 400; }
  .file.active { border-color: var(--accent); background: var(--hover); font-weight: 500; }
  .none { font-size: 14px; }
  .new { display: flex; gap: 8px; }
  .new input { flex: 1; min-width: 0; }
  .link { border: 0; background: transparent; height: auto; padding: 0; text-align: left; font-size: 14px; font-weight: 500; color: var(--accent); }
  .link:hover:not(:disabled) { background: transparent; text-decoration: underline; }
  .link.on { text-decoration: underline; }
  .content { flex: 999 1 640px; min-width: 0; display: flex; flex-direction: column; gap: 16px; }
  .title { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
  .title h2 { font-size: 20px; font-weight: 600; }
  .note { font-size: 13px; font-weight: 500; }
  .code { width: 100%; border-radius: 12px; background: var(--code); font-family: var(--mono); font-size: 13px; line-height: 1.7; padding: 14px 16px; white-space: pre; overflow: auto; resize: vertical; }
  .ok { font-size: 14px; font-weight: 500; display: flex; align-items: center; gap: 8px; }
  .problems { margin: 0; padding: 0; list-style: none; color: var(--bad); font-size: 14px; display: flex; flex-direction: column; gap: 4px; }
  .actions { display: flex; gap: 10px; flex-wrap: wrap; }
  .review { overflow: hidden; }
  .review-head { padding: 14px 18px; border-bottom: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
  .review-head h3 { font-size: 15px; font-weight: 600; }
  .diff { font-family: var(--mono); font-size: 13px; line-height: 1.7; overflow-x: auto; max-height: 420px; overflow-y: auto; }
  .line { padding: 0 18px; white-space: pre; color: var(--muted); }
  .line.add { background: color-mix(in srgb, var(--good) 18%, transparent); color: var(--text); }
  .line.del { background: color-mix(in srgb, var(--bad) 18%, transparent); color: var(--text); }
  .review-foot { padding: 14px 18px; border-top: 1px solid var(--border); display: flex; gap: 10px; flex-wrap: wrap; }
  .msg { gap: 8px; font-size: 14px; }
  .history { padding: 4px 18px; }
  .history summary { cursor: pointer; padding: 12px 0; font-size: 14px; font-weight: 600; }
  .hrow { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 0; border-top: 1px solid var(--border); font-size: 14px; flex-wrap: wrap; }
  .btns { display: flex; gap: 8px; }
  .head { gap: 6px; }
  .head h2 { font-size: 24px; font-weight: 600; }
  .head p { font-size: 14px; }
  .cols { display: flex; flex-wrap: wrap; gap: 24px; align-items: flex-start; }
  .form { flex: 1 1 380px; min-width: 0; padding: 24px; display: flex; flex-direction: column; gap: 18px; }
  .two { display: flex; flex-wrap: wrap; gap: 16px; }
  .two .field { flex: 1 1 140px; min-width: 0; }
  .field em { font-style: normal; font-weight: 400; }
  .check { display: flex; align-items: center; gap: 10px; font-size: 14px; cursor: pointer; }
  .preview { flex: 1 1 420px; min-width: 0; gap: 12px; }
  .row-between { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
  .row-between h3 { font-size: 15px; font-weight: 600; }
  .row-between h3 .mono { font-weight: 500; }
  .help { font-size: 13px; line-height: 1.5; }
  @media (max-width: 700px) { nav { flex-basis: 100%; } }
</style>
