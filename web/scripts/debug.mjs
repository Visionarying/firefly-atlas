import { chromium } from 'playwright-core'
const b = await chromium.launch({ executablePath: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe', headless: true })
const login = await fetch('http://127.0.0.1:3002/api/v1/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:'admin',password:process.env.ADMIN_PASSWORD||''})}).then(r=>r.json())
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(t=>localStorage.setItem('firefly_token',t), login.data.token)
const p = await ctx.newPage()
await p.goto('http://127.0.0.1:3002/#/', { waitUntil: 'networkidle' })
await p.waitForTimeout(800)
console.log(JSON.stringify(await p.evaluate(()=>{
  const st = el => { const c=getComputedStyle(el); return {tag:el.tagName,cls:el.className?.toString().slice(0,60),bg:c.backgroundColor,color:c.color,opacity:c.opacity,visibility:c.visibility,display:c.display,z:c.zIndex,transform:c.transform} }
  return {
    html: st(document.documentElement),
    body: st(document.body),
    app: st(document.getElementById('app')),
    home: document.querySelector('.home')?st(document.querySelector('.home')):null,
    text: document.body.innerText.slice(0,200),
    stack: document.elementsFromPoint(720,450).slice(0,8).map(st),
    sheets: [...document.styleSheets].map(s=>s.href||'inline').slice(0,5)
  }
}),null,2))
await b.close()
