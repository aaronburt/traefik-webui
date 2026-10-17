<script>
  import { get } from './api.js'

  let { onerror } = $props()
  let entries = $state([])
  let loading = $state(true)
  let error = $state('')

  async function load() {
    loading = true
    try {
      entries = await get('/api/audit')
      error = ''
    } catch (e) {
      if (e.unauthorized) return onerror(e)
      error = e.message
    } finally {
      loading = false
    }
  }
  load()
</script>

<div class="head">
  <div class="title">
    <h1>Audit log</h1>
    <span class="muted note">Records of administrative actions, authentication events, and configuration writes.</span>
  </div>
  <div class="actions">
    <a class="button dl" href="/api/audit/download" download="audit.csv">Download CSV</a>
    <button onclick={load}>Refresh</button>
  </div>
</div>

{#if error}
  <p class="bad" role="alert">Cannot load audit log: {error}</p>
{:else if loading && !entries.length}
  <p class="muted">Loading…</p>
{:else if !entries.length}
  <div class="card empty">
    <p class="muted">No audit events recorded yet.</p>
  </div>
{:else}
  <div class="card scroll">
    <table>
      <thead>
        <tr>
          <th>Time</th>
          <th>Event</th>
          <th>User</th>
          <th>Source IP</th>
          <th>Details</th>
        </tr>
      </thead>
      <tbody>
        {#each entries as e}
          <tr>
            <td class="mono muted">{new Date(e.timestamp).toLocaleString()}</td>
            <td><span class="badge {e.event.startsWith('auth_') ? 'auth' : 'config'}">{e.event}</span></td>
            <td>{e.user}</td>
            <td class="mono muted">{e.sourceIP}</td>
            <td>{e.details}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<style>
  .head { display: flex; align-items: flex-start; justify-content: space-between; flex-wrap: wrap; gap: 16px; }
  .title { display: flex; flex-direction: column; gap: 6px; }
  .title h1 { font-size: 24px; font-weight: 600; }
  .note { font-size: 14px; }
  .actions { display: flex; gap: 10px; align-items: center; }
  .button.dl { display: inline-flex; align-items: center; justify-content: center; height: 40px; padding: 0 16px; border-radius: 8px; border: 1px solid var(--border); background: var(--surface); font-weight: 500; font-size: 14px; text-decoration: none; color: inherit; }
  .button.dl:hover { background: var(--hover); }
  .scroll { overflow-x: auto; }
  .empty { padding: 32px; text-align: center; }
  table { border-collapse: collapse; width: 100%; font-size: 14px; }
  th { text-align: left; padding: 12px 16px; font-weight: 500; color: var(--muted); font-size: 12px; letter-spacing: 0.05em; text-transform: uppercase; }
  td { padding: 12px 16px; border-top: 1px solid var(--border); white-space: nowrap; }
  td:last-child { white-space: normal; word-break: break-word; }
  .badge { display: inline-block; padding: 2px 8px; border-radius: 6px; font-size: 12px; font-family: var(--mono); font-weight: 500; border: 1px solid var(--border); background: var(--hover); }
  .badge.config { border-color: var(--accent); color: var(--accent); }
  .badge.auth { border-color: var(--good); color: var(--good); }
</style>
