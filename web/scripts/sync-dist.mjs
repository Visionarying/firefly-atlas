// Copy the Vite build into the Go embed directory.
//
// The Go binary embeds service/router/webdist, so `go build` fails with
// "contains no embeddable files" when this step is skipped.
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

// fs.cpSync terminates the Node process on some Windows + Node 22 combinations
// (exit code 127 with no JavaScript stack trace), which silently leaves an empty
// webdist behind. Copying entry by entry keeps the step reliable everywhere.
function copyTree(src, dst, counter) {
  fs.mkdirSync(dst, { recursive: true })
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const from = path.join(src, entry.name)
    const to = path.join(dst, entry.name)
    if (entry.isDirectory()) {
      copyTree(from, to, counter)
    } else {
      fs.copyFileSync(from, to)
      counter.files += 1
    }
  }
}

fs.rmSync(target, { recursive: true, force: true })

const counter = { files: 0 }
copyTree(dist, target, counter)

if (!fs.existsSync(path.join(target, 'index.html'))) {
  console.error('copy finished but webdist/index.html is missing')
  process.exit(1)
}

console.log(`copied ${counter.files} files -> ${target}`)
