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

<form onsubmit={submit}>
  <h1>{setup ? 'Create admin password' : 'Sign in'}</h1>
  {#if setup}
    <p>First run: choose a password of at least 12 characters.</p>
  {/if}
  <label>
    Password
    <input type="password" bind:value={password} autocomplete={setup ? 'new-password' : 'current-password'} required minlength={setup ? 12 : undefined} />
  </label>
  {#if setup}
    <label>
      Confirm password
      <input type="password" bind:value={confirm} autocomplete="new-password" required />
    </label>
  {/if}
  {#if error}<p role="alert" class="error">{error}</p>{/if}
  <button disabled={busy}>{setup ? 'Create password' : 'Sign in'}</button>
</form>

<style>
  form { display: grid; gap: 1rem; max-width: 22rem; }
  label { display: grid; gap: 0.25rem; }
  input, button { font: inherit; padding: 0.5rem; }
  .error { color: #b00020; margin: 0; }
</style>
