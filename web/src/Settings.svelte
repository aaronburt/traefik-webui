<script>
  import { get, post } from './api.js'

  let { onerror } = $props()

  let oldPassword = $state('')
  let newPassword = $state('')
  let confirmPassword = $state('')
  let error = $state('')
  let success = $state('')
  let busy = $state(false)

  let version = $state(null)
  let overview = $state(null)

  async function loadSystem() {
    // Traefik being down is shown as "Unavailable"; an expired session is not.
    const soft = (e) => {
      if (e.unauthorized) throw e
      return null
    }
    try {
      ;[version, overview] = await Promise.all([get('/api/traefik/version').catch(soft), get('/api/traefik/overview').catch(soft)])
    } catch (e) {
      if (e.unauthorized) onerror(e)
    }
  }
  loadSystem()

  async function updatePassword(e) {
    e.preventDefault()
    error = ''
    success = ''

    if (newPassword !== confirmPassword) {
      error = 'New passwords do not match'
      return
    }
    if (newPassword.length < 12) {
      error = 'New password must be at least 12 characters'
      return
    }

    busy = true
    try {
      await post('/api/auth/password', { oldPassword, newPassword })
      oldPassword = ''
      newPassword = ''
      confirmPassword = ''
      success = 'Password successfully updated'
    } catch (err) {
      if (err.unauthorized) return onerror(err)
      error = err.message
    } finally {
      busy = false
    }
  }
</script>

<div class="head">
  <h1>Settings</h1>
  <span class="muted note">Account security and system configuration.</span>
</div>

<div class="grid">
  <section class="card pane">
    <div class="pane-head">
      <h2>Change admin password</h2>
      <p class="muted">Update the local administrator account password. Minimum 12 characters.</p>
    </div>

    <form class="form" onsubmit={updatePassword}>
      <label class="field">
        <span>Current password</span>
        <input type="password" bind:value={oldPassword} required autocomplete="current-password" />
      </label>

      <label class="field">
        <span>New password</span>
        <input type="password" bind:value={newPassword} required minlength="12" maxlength="72" autocomplete="new-password" />
      </label>

      <label class="field">
        <span>Confirm new password</span>
        <input type="password" bind:value={confirmPassword} required minlength="12" maxlength="72" autocomplete="new-password" />
      </label>

      {#if error}
        <p class="bad alert" role="alert">
          <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" stroke-width="1.5" /><path d="M8 4.5v4M8 10.8v.4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" /></svg>
          {error}
        </p>
      {/if}

      {#if success}
        <p class="good alert" role="status">
          <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" stroke-width="1.5" /><path d="M5 8.2l2 2 4-4.4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
          {success}
        </p>
      {/if}

      <button class="primary" disabled={busy || !oldPassword || !newPassword || !confirmPassword}>
        {busy ? 'Updating…' : 'Update password'}
      </button>
    </form>
  </section>

  <section class="card pane">
    <div class="pane-head">
      <h2>System overview</h2>
      <p class="muted">Connected environment and runtime status.</p>
    </div>

    <div class="stack rows">
      <div class="row">
        <span class="muted">Connected Traefik version</span>
        <span class="mono">{version?.Version ?? 'Unavailable'}</span>
      </div>
      {#if version?.Codename}
        <div class="row">
          <span class="muted">Codename</span>
          <span>{version.Codename}</span>
        </div>
      {/if}
      {#if version?.startDate}
        <div class="row">
          <span class="muted">Traefik started</span>
          <span>{new Date(version.startDate).toLocaleString()}</span>
        </div>
      {/if}
      <div class="row">
        <span class="muted">Active providers</span>
        <span class="pills">
          {#each overview?.providers ?? [] as p}
            <span class="pill">{p}</span>
          {:else}
            <span class="muted">none</span>
          {/each}
        </span>
      </div>
      <div class="row">
        <span class="muted">Authentication</span>
        <span>Local single-user (bcrypt)</span>
      </div>
      <div class="row">
        <span class="muted">Session duration</span>
        <span>12 hours (HttpOnly, Strict)</span>
      </div>
    </div>
  </section>
</div>

<style>
  .head { display: flex; flex-direction: column; gap: 6px; }
  .head h1 { font-size: 24px; font-weight: 600; }
  .note { font-size: 14px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(340px, 1fr)); gap: 24px; align-items: start; }
  .pane { padding: 24px; display: flex; flex-direction: column; gap: 20px; }
  .pane-head { display: flex; flex-direction: column; gap: 6px; }
  .pane-head h2 { font-size: 18px; font-weight: 600; }
  .pane-head p { font-size: 14px; line-height: 1.5; }
  .form { display: flex; flex-direction: column; gap: 16px; }
  .alert { font-size: 13px; display: flex; align-items: center; gap: 8px; }
  .rows { gap: 12px; font-size: 14px; }
  .row { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding-bottom: 10px; border-bottom: 1px solid var(--border); }
  .row:last-child { border-bottom: none; padding-bottom: 0; }
  .pills { display: flex; flex-wrap: wrap; gap: 6px; justify-content: flex-end; }
</style>
