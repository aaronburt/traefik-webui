// A tiny stand-in for Traefik's API, so the UI can be tested without a real Traefik.
const http = require('http')

const file = (name) => ({ name: `${name}@file`, status: 'enabled', provider: 'file' })
const data = {
  '/api/version': { Version: '3.0.0', Codename: 'mock' },
  '/api/overview': {
    http: { routers: { total: 2 }, services: { total: 2 }, middlewares: { total: 0 } },
    tcp: { routers: { total: 0 }, services: { total: 0 }, middlewares: { total: 0 } },
    udp: { routers: { total: 0 }, services: { total: 0 } },
    features: {},
    providers: ['file', 'internal'],
  },
  '/api/entrypoints': [
    { name: 'web', address: ':80' },
    { name: 'websecure', address: ':443' },
  ],
  '/api/http/routers': [
    { ...file('e2e-app'), rule: 'Host(`e2e.example.com`)', service: 'e2e-app@file', entryPoints: ['web'] },
    { name: 'dashboard@internal', status: 'enabled', provider: 'internal', rule: 'PathPrefix(`/dashboard`)', service: 'api@internal' },
  ],
  '/api/http/services': [{ ...file('e2e-app'), type: 'loadbalancer', loadBalancer: { servers: [{ url: 'http://127.0.0.1:9' }] } }],
  '/api/http/middlewares': [],
  '/api/tcp/routers': [],
  '/api/tcp/services': [],
  '/api/tcp/middlewares': [],
  '/api/udp/routers': [],
  '/api/udp/services': [],
}

http
  .createServer((req, res) => {
    const body = data[new URL(req.url, 'http://x').pathname]
    res.writeHead(body ? 200 : 404, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify(body ?? { message: 'not found' }))
  })
  .listen(Number(process.env.MOCK_PORT), '127.0.0.1')
