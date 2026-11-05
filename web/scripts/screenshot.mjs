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

async function withPage(viewport, fn) {
  const context = await browser.newContext({ viewport, deviceScaleFactor: 1 })
  await context.addInitScript((token) => localStorage.setItem('firefly_token', token), login.data.token)
  const page = await context.newPage()
  await page.goto(`${BASE}/#/`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(600)
  // Normalize to light theme regardless of the persisted setting.
  if (await page.getByTitle('切换浅色').count()) {
    await page.getByTitle('切换浅色').click()
    await page.waitForTimeout(800)
  }
  await fn(page)
  await context.close()
}

await withPage({ width: 1440, height: 900 }, async (page) => {
  await page.screenshot({ path: `${OUT}\\desktop-home-light.png` })
  await page.getByTitle('切换深色').click()
  await page.waitForTimeout(800)
  await page.screenshot({ path: `${OUT}\\desktop-home-dark.png` })
  await page.getByTitle('切换浅色').click()
  await page.waitForTimeout(900)
})

await withPage({ width: 1440, height: 900 }, async (page) => {
  await page.getByTitle('系统设置').click()
  await page.getByText('风格设置', { exact: true }).click()
  await page.waitForTimeout(400)
  await page.screenshot({ path: `${OUT}\\desktop-settings.png` })
  await page.getByText('分组管理', { exact: true }).click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: `${OUT}\\desktop-settings-groups.png` })
  await page.getByText('账号管理', { exact: true }).click()
  await page.waitForTimeout(600)
  await page.screenshot({ path: `${OUT}\\desktop-settings-account.png` })
})

await withPage({ width: 1440, height: 900 }, async (page) => {
  await page.getByTitle('编辑布局').click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: `${OUT}\\desktop-edit.png` })
  await page.getByTitle('完成编辑').click()
})

await withPage({ width: 390, height: 844 }, async (page) => {
  await page.screenshot({ path: `${OUT}\\mobile-home.png` })
  await page.getByTitle('系统设置').click()
  await page.waitForTimeout(400)
  await page.screenshot({ path: `${OUT}\\mobile-settings.png` })
})

await browser.close()
console.log('done')
