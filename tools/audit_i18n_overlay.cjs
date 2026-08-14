#!/usr/bin/env node

const fs = require('fs')
const path = require('path')
const vm = require('vm')
const crypto = require('crypto')
const { execFileSync, spawnSync } = require('child_process')
const ts = require('../frontend/node_modules/typescript')

const repoRoot = path.resolve(__dirname, '..')
const localesRoot = path.join(repoRoot, 'frontend/src/i18n/locales')
const customLocalesRoot = path.join(repoRoot, 'frontend/src/custom/i18n/locales')
const check = process.argv.includes('--check')
const refArg = process.argv.find((arg) => arg.startsWith('--upstream='))
const upstreamRef = refArg ? refArg.slice('--upstream='.length) : 'upstream/main'
const policyArg = process.argv.find((arg) => arg.startsWith('--policy='))
const policyPath = policyArg
  ? path.resolve(repoRoot, policyArg.slice('--policy='.length))
  : path.join(repoRoot, 'tools/i18n_overlay_policy.json')

function loadModule(file, cache = new Map()) {
  file = path.resolve(file)
  if (cache.has(file)) return cache.get(file).exports
  const source = fs.readFileSync(file, 'utf8')
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2020,
      esModuleInterop: true,
    },
  }).outputText
  const module = { exports: {} }
  cache.set(file, module)
  const dirname = path.dirname(file)
  const localRequire = (specifier) => {
    if (!specifier.startsWith('.')) return require(specifier)
    let resolved = path.resolve(dirname, specifier)
    if (fs.existsSync(`${resolved}.ts`)) resolved = `${resolved}.ts`
    else if (fs.existsSync(path.join(resolved, 'index.ts'))) resolved = path.join(resolved, 'index.ts')
    else throw new Error(`Cannot resolve ${specifier} from ${file}`)
    return loadModule(resolved, cache)
  }
  vm.runInNewContext(
    output,
    { require: localRequire, module, exports: module.exports, __dirname: dirname, __filename: file },
    { filename: file },
  )
  return module.exports
}

function loadDefault(file) {
  const loaded = loadModule(file)
  return loaded.default || loaded
}

function flatten(value, prefix = '', result = {}) {
  for (const [key, child] of Object.entries(value || {})) {
    const fullKey = prefix ? `${prefix}.${key}` : key
    if (child && typeof child === 'object' && !Array.isArray(child)) {
      flatten(child, fullKey, result)
    } else {
      result[fullKey] = child
    }
  }
  return result
}

function digest(value) {
  return crypto.createHash('sha256').update(JSON.stringify(value)).digest('hex')
}

function sortedEntries(value) {
  return Object.entries(value).sort(([left], [right]) => left.localeCompare(right))
}

function shapeConflicts(base, overlay, prefix = '', result = []) {
  for (const [key, child] of Object.entries(overlay || {})) {
    if (!(key in (base || {}))) continue
    const baseChild = base[key]
    const fullKey = prefix ? `${prefix}.${key}` : key
    const baseObject = baseChild && typeof baseChild === 'object' && !Array.isArray(baseChild)
    const overlayObject = child && typeof child === 'object' && !Array.isArray(child)
    if (baseObject !== overlayObject) {
      result.push(fullKey)
      continue
    }
    if (baseObject && overlayObject) shapeConflicts(baseChild, child, fullKey, result)
  }
  return result
}

function loadUpstreamLocale(locale) {
  const tempRoot = fs.mkdtempSync(path.join('/tmp', `sub2api-i18n-audit-${locale}-`))
  const prefix = `frontend/src/i18n/locales/${locale}/`
  const files = execFileSync(
    'git',
    ['ls-tree', '-r', '--name-only', upstreamRef, prefix],
    { cwd: repoRoot, encoding: 'utf8' },
  ).trim().split('\n').filter(Boolean)
  try {
    for (const repoPath of files) {
      const target = path.join(tempRoot, repoPath.slice(prefix.length))
      fs.mkdirSync(path.dirname(target), { recursive: true })
      fs.writeFileSync(
        target,
        execFileSync('git', ['show', `${upstreamRef}:${repoPath}`], {
          cwd: repoRoot,
          encoding: 'utf8',
        }),
      )
    }
    return loadDefault(path.join(tempRoot, 'index.ts'))
  } finally {
    fs.rmSync(tempRoot, { recursive: true, force: true })
  }
}

function officialModulePaths(locale) {
  const prefix = `frontend/src/i18n/locales/${locale}/`
  return execFileSync('git', ['ls-tree', '-r', '--name-only', upstreamRef, prefix], {
    cwd: repoRoot,
    encoding: 'utf8',
  }).trim().split('\n').filter(Boolean)
}

let policy = null
try {
  policy = JSON.parse(fs.readFileSync(policyPath, 'utf8'))
} catch (error) {
  console.error(`cannot load i18n overlay policy ${policyPath}: ${error.message}`)
}

if (policy && policy.version !== 1) {
  console.error(`unsupported i18n overlay policy version: ${policy.version}`)
}
let failed = !policy || policy.version !== 1
for (const locale of ['zh', 'en']) {
  const customTree = loadDefault(path.join(customLocalesRoot, `${locale}-custom.ts`))
  const upstreamTree = loadUpstreamLocale(locale)
  const effectiveTree = loadDefault(path.join(customLocalesRoot, `${locale}.ts`))
  const custom = flatten(customTree)
  const upstream = flatten(upstreamTree)
  const effective = flatten(effectiveTree)
  const keys = Object.keys(custom).sort()
  const added = keys.filter((key) => !(key in upstream))
  const redundant = keys.filter(
    (key) => key in upstream && JSON.stringify(custom[key]) === JSON.stringify(upstream[key]),
  )
  const overrides = keys.filter(
    (key) => key in upstream && JSON.stringify(custom[key]) !== JSON.stringify(upstream[key]),
  )
  const expectedEffective = { ...upstream, ...custom }
  const effectiveMatches = JSON.stringify(sortedEntries(effective)) === JSON.stringify(sortedEntries(expectedEffective))
  const conflicts = shapeConflicts(upstreamTree, customTree)
  const addedSignature = digest(added)
  const overrideSignature = digest(overrides.map((key) => [key, upstream[key]]))
  const expectedPolicy = policy?.locales?.[locale]
  const policyMatches = Boolean(
    expectedPolicy &&
    expectedPolicy.added_keys === added.length &&
    expectedPolicy.overrides === overrides.length &&
    expectedPolicy.added_keys_sha256 === addedSignature &&
    expectedPolicy.override_upstream_values_sha256 === overrideSignature
  )
  const officialPaths = officialModulePaths(locale)
  const diff = spawnSync('git', ['diff', '--quiet', upstreamRef, '--', ...officialPaths], {
    cwd: repoRoot,
  })
  const officialModulesMatch = diff.status === 0

  console.log(
    `${locale}: custom=${keys.length} added=${added.length} overrides=${overrides.length} ` +
      `redundant=${redundant.length} official_modules_match=${officialModulesMatch} ` +
      `effective_merge_match=${effectiveMatches} policy_match=${policyMatches} ` +
      `added_sig=${addedSignature} override_sig=${overrideSignature}`,
  )
  if (!officialModulesMatch) {
    console.error(`${locale}: official locale domain files differ from ${upstreamRef}`)
    failed = true
  }
  if (redundant.length > 0) {
    console.error(`${locale}: redundant custom keys: ${redundant.slice(0, 20).join(', ')}`)
    failed = true
  }
  if (!effectiveMatches) {
    console.error(`${locale}: effective locale is not the official locale plus custom overlay`)
    failed = true
  }
  if (conflicts.length > 0) {
    console.error(`${locale}: overlay object/value shape conflicts: ${conflicts.slice(0, 20).join(', ')}`)
    failed = true
  }
  if (!policyMatches) {
    console.error(`${locale}: overlay key/upstream-value signature changed; review and update ${policyPath}`)
    failed = true
  }
}

if (check && failed) process.exit(2)
