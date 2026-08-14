import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'
import { transformWegooIndexHtml } from '@/custom/vite/wegooIndexHtml'

const indexSource = readFileSync(resolve(__dirname, '../../../index.html'), 'utf8')
const mainSource = readFileSync(resolve(__dirname, '../main.ts'), 'utf8')
const styleSource = readFileSync(resolve(__dirname, '../styles/style.css'), 'utf8')

describe('site style overlay', () => {
  it('loads the reviewed site stylesheet from the custom overlay', () => {
    expect(indexSource).toContain('src="/src/main.ts"')
    expect(transformWegooIndexHtml(indexSource)).toContain('src="/src/custom/main.ts"')
    expect(mainSource).toContain("import '@/custom/styles/style.css'")
    expect(styleSource).toContain('@tailwind base;')
    expect(styleSource).toContain('--gw-bg:')
    expect(styleSource).toContain('--apple-blue:')
  })
})
