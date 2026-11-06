import { chromium } from 'playwright-core'

const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'
const OUT = '.refs'

const browser = await chromium.launch({ executablePath: CHROME, headless: true })
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await context.newPage()
const logs = []
page.on('console', (m) => logs.push(`${m.type()}: ${m.text()}`))
page.on('pageerror', (e) => logs.push(`PAGEERROR: ${e.message}`))
await page.goto('http://127.0.0.1:3002/#/', { waitUntil: 'networkidle' })
await page.waitForTimeout(800)
await page.screenshot({ path: `${OUT}\\guest-home.png` })
const info = await page.evaluate(() => ({
  hasSettings: !!document.querySelector('[title="系统设置"]'),
  hasEdit: !!document.querySelector('[title="编辑布局"]'),
  hasLogin: [...document.querySelectorAll('button')].some((b) => b.textContent?.includes('登录')),
  cards: document.querySelectorAll('.nav-card').length,
  groups: document.querySelectorAll('.group').length,
}))
console.log(JSON.stringify({ info, logs: logs.filter((l) => !l.startsWith('debug:')) }, null, 2))
await browser.close()
