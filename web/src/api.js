// Every write sends this header; the backend rejects unsafe requests without it.
const headers = { 'Content-Type': 'application/json', 'X-Requested-With': 'traefik-webui' }

export async function status() {
  const res = await fetch('/api/auth/status')
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

export async function post(path, body) {
  const res = await fetch(path, { method: 'POST', headers, body: JSON.stringify(body ?? {}) })
  if (!res.ok) throw new Error((await res.text()).trim() || `HTTP ${res.status}`)
  return res.json()
}

export async function get(path) {
  const res = await fetch(path)
  if (res.status === 401) throw Object.assign(new Error('unauthorized'), { unauthorized: true })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}
