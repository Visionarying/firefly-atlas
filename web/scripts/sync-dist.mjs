// Copy the Vite build into the Go embed directory.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')
const dist = path.join(root, 'dist')
const target = path.resolve(root, '..', 'service', 'router', 'webdist')

if (!fs.existsSync(path.join(dist, 'index.html'))) {
  console.error('dist/index.html not found, run vite build first')
  process.exit(1)
}

fs.rmSync(target, { recursive: true, force: true })
fs.mkdirSync(target, { recursive: true })
fs.cpSync(dist, target, { recursive: true })
console.log(`copied ${dist} -> ${target}`)
