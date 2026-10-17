<script>
  import { post } from './api.js'

  let { setup, onsuccess } = $props()
  let password = $state('')
  let confirm = $state('')
  let error = $state('')
  let busy = $state(false)

  async function submit(e) {
    e.preventDefault()
    error = ''
    if (setup && password !== confirm) {
      error = 'Passwords do not match'
      return
    }
    busy = true
    try {
      await post(setup ? '/api/auth/setup' : '/api/auth/login', { password })
      password = confirm = ''
      onsuccess()
    } catch (err) {
      error = err.message
    } finally {
      busy = false
    }
  }
</script>

<form class="card" onsubmit={submit}>
  <div class="head">
    <h1>{setup ? 'Create admin password' : 'Sign in'}</h1>
    <p class="muted">
      {setup ? 'First run: choose a password of at least 12 characters.' : 'Manage your Traefik routes, services and middlewares.'}
    </p>
  </div>
  <label class="field">
    <span>Password</span>
    <input type="password" bind:value={password} autocomplete={setup ? 'new-password' : 'current-password'} required minlength={setup ? 12 : undefined} />
  </label>
  {#if setup}
    <label class="field">
      <span>Confirm password</span>
      <input type="password" bind:value={confirm} autocomplete="new-password" required />
    </label>
  {/if}
  {#if error}
    <p role="alert" class="error bad">
      <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" stroke-width="1.5" /><path d="M8 4.5v4M8 10.8v.4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" /></svg>
      {error}
    </p>
  {/if}
  <button class="primary" disabled={busy}>{setup ? 'Create password' : 'Sign in'}</button>
</form>

<style>
  form { width: 100%; max-width: 400px; padding: 32px; display: flex; flex-direction: column; gap: 20px; }
  .head { display: flex; flex-direction: column; gap: 6px; }
  h1 { font-size: 24px; font-weight: 600; }
  .head p { font-size: 14px; line-height: 1.5; }
  .error { font-size: 13px; display: flex; gap: 8px; align-items: center; }
  .field input { height: 44px; background: var(--bg); font-size: 15px; }
  button { height: 44px; font-size: 15px; }
</style>
