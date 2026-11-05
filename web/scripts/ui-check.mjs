import { chromium } from 'playwright-core'

const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'
const BASE = 'http://127.0.0.1:3002'
const OUT = 'D:\\ai\\opencode\\工作区\\firefly-panel\\.refs'

const login = await fetch(`${BASE}/api/v1/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: process.env.ADMIN_PASSWORD || '' }),
}).then((r) => r.json())
if (login.code !== 0) throw new Error('login failed')

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
if (await page.getByTitle('切换浅色').count()) {
  await page.getByTitle('切换浅色').click()
  await page.waitForTimeout(700)
}

// Edit mode + edit-card modal
await page.getByTitle('编辑布局').click()
await page.waitForTimeout(300)
const firstCard = page.locator('.nav-card').first()
await firstCard.hover()
await firstCard.locator('button[title="编辑"]').click()
await page.waitForTimeout(500)
const editModalVisible = await page.getByText('编辑卡片', { exact: true }).count()
await page.screenshot({ path: `${OUT}\\edit-modal.png` })
await page.getByTitle('关闭').click()
await page.waitForTimeout(300)

// New group prompt
await page.getByRole('button', { name: '新建分组' }).last().click()
await page.waitForTimeout(400)
const promptVisible = await page.getByText('新建分组', { exact: true }).count()
await page.screenshot({ path: `${OUT}\\group-prompt.png` })
await page.getByRole('button', { name: '取消' }).click()
await page.waitForTimeout(300)
await page.getByTitle('完成编辑').click()
await page.waitForTimeout(400)

console.log(JSON.stringify({ editModalVisible, promptVisible, logs }, null, 2))
await browser.close()
