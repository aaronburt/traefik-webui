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

<div class="auth">
  <form class="card" onsubmit={submit}>
    <h1>{setup ? 'Create admin password' : 'Sign in'}</h1>
    <p class="muted">
      {setup ? 'First run: choose a password of at least 12 characters.' : 'Manage your Traefik routes, services and middlewares.'}
    </p>
    <label>
      <span>Password</span>
      <input type="password" bind:value={password} autocomplete={setup ? 'new-password' : 'current-password'} required minlength={setup ? 12 : undefined} />
    </label>
    {#if setup}
      <label>
        <span>Confirm password</span>
        <input type="password" bind:value={confirm} autocomplete="new-password" required />
      </label>
    {/if}
    {#if error}<p role="alert" class="bad">{error}</p>{/if}
    <button class="primary" disabled={busy}>{setup ? 'Create password' : 'Sign in'}</button>
  </form>
</div>

<style>
  .auth {
    display: flex;
    justify-content: center;
    padding-top: 48px;
  }
  form {
    display: flex;
    flex-direction: column;
    gap: 18px;
    width: 100%;
    max-width: 400px;
    padding: 32px;
  }
  h1 {
    margin: 0;
    font-size: 24px;
  }
  p {
    margin: 0;
    font-size: 14px;
    line-height: 1.5;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 13px;
    font-weight: 500;
  }
  label input {
    height: 44px;
    font-size: 15px;
  }
  button.primary {
    height: 44px;
    width: 100%;
    font-size: 15px;
  }
</style>
