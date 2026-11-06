import { chromium } from 'playwright-core'

const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'
const BASE = 'http://127.0.0.1:3002'
const OUT = '.refs'

const login = await fetch(`${BASE}/api/v1/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: process.env.ADMIN_PASSWORD || '' }),
}).then((r) => r.json())

const browser = await chromium.launch({ executablePath: CHROME, headless: true })
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await context.addInitScript((token) => localStorage.setItem('firefly_token', token), login.data.token)
const page = await context.newPage()
const logs = []
page.on('console', (m) => {
  if (m.type() === 'error' || m.type() === 'warning') logs.push(`${m.type()}: ${m.text()}`)
})
page.on('pageerror', (e) => logs.push(`PAGEERROR: ${e.message}`))
await page.goto(`${BASE}/#/`, { waitUntil: 'networkidle' })
await page.waitForTimeout(600)
await page.getByTitle('系统设置').click()
await page.getByText('风格设置', { exact: true }).click()
await page.waitForTimeout(400)
await page.getByText('搜索引擎', { exact: true }).scrollIntoViewIfNeeded()
await page.waitForTimeout(300)
await page.screenshot({ path: `${OUT}\\desktop-settings-search.png` })
console.log(JSON.stringify({ logs }, null, 2))
await browser.close()
