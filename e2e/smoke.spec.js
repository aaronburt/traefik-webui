const { test, expect } = require('@playwright/test')

const password = 'correct horse battery staple'
const appYaml = [
  'http:',
  '  routers:',
  '    e2e-app:',
  '      rule: "Host(`e2e.example.com`)"',
  '      service: e2e-app',
  '  services:',
  '    e2e-app:',
  '      loadBalancer:',
  '        servers:',
  '          - url: http://127.0.0.1:9',
  '',
].join('\n')

test.describe.configure({ mode: 'serial' })

let page
test.beforeAll(async ({ browser }) => {
  page = await browser.newPage()
})
test.afterAll(async () => page.close())

test('first run creates the admin password and signs in', async () => {
  await page.goto('/')
  await page.getByLabel('Password', { exact: true }).fill(password)
  await page.getByLabel('Confirm password').fill(password)
  await page.getByRole('button', { name: 'Create password' }).click()
  await expect(page.getByRole('navigation', { name: 'Pages' })).toBeVisible()
  await expect(page.getByText('Connected to Traefik 3.0.0')).toBeVisible()
})

test('objects tab lists what Traefik reports', async () => {
  await page.getByRole('link', { name: 'Objects' }).click()
  await expect(page.getByRole('button', { name: 'e2e-app@file' }).first()).toBeVisible()
  await page.getByRole('link', { name: /Entrypoints/ }).click()
  await expect(page.getByRole('button', { name: 'websecure' })).toBeVisible()
})

test('config editor validates, reviews and applies a file', async () => {
  await page.getByRole('link', { name: 'Config files' }).click()
  await page.getByLabel('New file name').fill('e2e')
  await page.getByRole('button', { name: 'New', exact: true }).click()

  const editor = page.getByLabel('YAML content')
  await editor.fill('http: [')
  await expect(page.getByRole('alert')).toContainText('invalid YAML')
  await expect(page.getByRole('button', { name: 'Review changes' })).toBeDisabled()

  await editor.fill(appYaml)
  await expect(page.getByText('No problems found.')).toBeVisible()
  await page.getByRole('button', { name: 'Review changes' }).click()
  await expect(page.getByLabel('Review')).toContainText('+ http:')

  // Editing after a review drops the diff, so Apply can only ever save what was reviewed.
  await editor.press('End')
  await editor.type('# edited')
  await expect(page.getByLabel('Review')).toHaveCount(0)

  await page.getByRole('button', { name: 'Review changes' }).click()
  await page.getByRole('button', { name: 'Confirm and apply' }).click()
  await expect(page.getByRole('status')).toContainText('Saved')
  await expect(page.getByRole('button', { name: /^e2e\.yml/ })).toBeVisible()
})

test('quick route takes entrypoints from Traefik and flags a clash', async () => {
  await page.getByRole('button', { name: 'Quick route form' }).click()
  const entrypoint = page.getByLabel('Entrypoint')
  await expect(entrypoint.locator('option')).toHaveText(['Choose one', 'web', 'websecure'])
  await expect(page.getByLabel('Route file YAML')).toHaveValue('')

  // The router name e2e-app already lives in e2e.yml, so a new route file may not reuse it.
  await page.getByPlaceholder('my-app').fill('e2e-app')
  await page.getByLabel('Domain').fill('other.example.com')
  await page.getByLabel('Backend URL').fill('http://127.0.0.1:9')
  await entrypoint.selectOption('websecure')
  await expect(page.getByRole('alert')).toContainText('already defined in another file')

  await page.getByPlaceholder('my-app').fill('fresh-app')
  await expect(page.getByText('No problems found.')).toBeVisible()
})

test('a wrong password is rejected and the right one signs back in', async () => {
  await page.getByRole('button', { name: 'Sign out' }).click()
  await page.getByLabel('Password').fill('not the password')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('navigation', { name: 'Pages' })).toHaveCount(0)

  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('navigation', { name: 'Pages' })).toBeVisible()
})
