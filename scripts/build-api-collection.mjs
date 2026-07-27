#!/usr/bin/env node
/**
 * Turns the backend's OpenAPI spec into the collection the API console loads.
 *
 * Why derive instead of hand-listing endpoints: a hand-written list rots the
 * moment a route changes, and the console would then lie about the API. The
 * spec is already drift-guarded by backend tests (apidocs/coverage_test.go), so
 * deriving from it means the console inherits that guarantee.
 *
 * The output is deliberately small (~30 KB vs the spec's 124 KB): the console
 * only needs the request side — method, path, params, a starter body — plus
 * enough prose to explain each call. Response schemas stay in /docs.
 *
 *   node scripts/build-api-collection.mjs
 */

import { readFileSync, writeFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const SPEC = resolve(root, 'backend/internal/apidocs/openapi.json')
const OUT = resolve(root, 'public/api-collection.json')

const spec = JSON.parse(readFileSync(SPEC, 'utf8'))

/** Follows a local `$ref` to its target. Only `#/`-style refs exist here. */
function deref(node) {
  if (!node || typeof node !== 'object' || !node.$ref) return node
  const parts = node.$ref.replace(/^#\//, '').split('/')
  let cur = spec
  for (const p of parts) cur = cur?.[p]
  return deref(cur)
}

/**
 * Builds a starter value for a schema. The goal is a body you can send after
 * filling in ids — not a full fixture — so optional fields are skipped unless
 * they carry a default.
 */
function sampleFor(schema, key = '', depth = 0) {
  const s = deref(schema)
  if (!s || depth > 4) return null
  if (s.allOf) return Object.assign({}, ...s.allOf.map((x) => sampleFor(x, key, depth) ?? {}))
  if (s.oneOf || s.anyOf) return sampleFor((s.oneOf ?? s.anyOf)[0], key, depth)
  if (s.default !== undefined) return s.default
  if (s.example !== undefined) return s.example
  if (s.enum?.length) return s.enum[0]

  switch (s.type) {
    case 'object': {
      const required = new Set(s.required ?? [])
      const out = {}
      for (const [name, prop] of Object.entries(s.properties ?? {})) {
        const p = deref(prop)
        if (!required.has(name) && p?.default === undefined) continue
        out[name] = sampleFor(prop, name, depth + 1)
      }
      return out
    }
    case 'array':
      return s.items ? [sampleFor(s.items, key, depth + 1)] : []
    case 'integer':
    case 'number':
      if (/amount|fee|price/i.test(key)) return 1_500_000
      return s.minimum ?? 0
    case 'boolean':
      return false
    default:
      if (s.format === 'date') return new Date().toISOString().slice(0, 10)
      if (s.format === 'date-time') return new Date().toISOString()
      if (s.format === 'uuid') return ''
      if (/email/i.test(key)) return 'nama@contoh.com'
      if (/password|sandi/i.test(key)) return ''
      return ''
  }
}

/**
 * Trims OpenAPI prose to one line of PLAIN text. Param hints render as bare
 * strings, so leaving `**` and backticks in would show the markup itself.
 */
function firstLine(text) {
  if (!text) return ''
  const line = text.split('\n').find((l) => l.trim()) ?? ''
  return line
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .trim()
}

/**
 * Flattens a request body's top-level properties into form fields.
 *
 * The console used to show one raw JSON textarea, which asks the reader to know
 * both the field names and JSON syntax before they can send anything. A labelled
 * field per property removes both requirements; the raw JSON stays one click
 * away for anyone who prefers it.
 *
 * Nested objects/arrays keep a small JSON box of their own — only the Paper.id
 * callback has them, and inventing a nested form for one endpoint would cost
 * more than it returns.
 */
function bodyFieldsFor(schema) {
  const s = deref(schema)
  if (!s) return undefined

  // allOf composes into one flat property set for our schemas.
  const merged = s.allOf
    ? s.allOf.map(deref).reduce(
        (acc, part) => ({
          properties: { ...acc.properties, ...(part.properties ?? {}) },
          required: [...acc.required, ...(part.required ?? [])],
        }),
        { properties: {}, required: [] },
      )
    : { properties: s.properties ?? {}, required: s.required ?? [] }

  const entries = Object.entries(merged.properties)
  if (entries.length === 0) return undefined

  const required = new Set(merged.required)
  return entries.map(([name, prop]) => {
    const p = deref(prop) ?? {}
    const complex = p.type === 'object' || p.type === 'array'
    return {
      name,
      // `title` is the human label. Derived labels get names like
      // "Send Whats App" — a camelCase splitter cannot know WhatsApp is one
      // word — so the readable version belongs in the spec with everything else.
      label: p.title,
      type: p.type ?? 'string',
      format: p.format,
      required: required.has(name),
      enum: p.enum,
      default: p.default,
      description: firstLine(p.description),
      complex,
    }
  })
}

const METHODS = ['get', 'post', 'put', 'patch', 'delete']
const operations = []

for (const [path, item] of Object.entries(spec.paths)) {
  const shared = (item.parameters ?? []).map(deref)
  for (const method of METHODS) {
    const op = item[method]
    if (!op) continue

    const params = [...shared, ...(op.parameters ?? []).map(deref)].map((p) => {
      const s = deref(p.schema) ?? {}
      return {
        name: p.name,
        in: p.in,
        required: Boolean(p.required),
        type: s.type ?? 'string',
        enum: s.enum,
        default: s.default,
        description: firstLine(p.description),
      }
    })

    const bodySchema = op.requestBody?.content?.['application/json']?.schema
    const multipart = op.requestBody?.content?.['multipart/form-data']

    operations.push({
      id: `${method}:${path}`,
      method: method.toUpperCase(),
      path,
      tag: op.tags?.[0] ?? 'Lainnya',
      summary: op.summary ?? `${method.toUpperCase()} ${path}`,
      description: op.description ?? '',
      params,
      body: bodySchema ? sampleFor(bodySchema) : undefined,
      bodyFields: bodySchema ? bodyFieldsFor(bodySchema) : undefined,
      bodyRequired: Boolean(op.requestBody?.required),
      multipart: Boolean(multipart),
      // `security: []` on an operation opts out of the global bearer default.
      auth: op.security ? op.security.length > 0 : true,
      // Prose is the only place the admin-only rule is written down.
      admin: /admin saja/i.test(op.description ?? ''),
    })
  }
}

// Sidebar order follows the spec's own tag order, so the console reads like the
// docs page rather than alphabetically scrambling related calls.
const tagOrder = (spec.tags ?? []).map((t) => t.name)
operations.sort((a, b) => {
  const ta = tagOrder.indexOf(a.tag)
  const tb = tagOrder.indexOf(b.tag)
  if (ta !== tb) return (ta < 0 ? 99 : ta) - (tb < 0 ? 99 : tb)
  return a.path.localeCompare(b.path) || a.method.localeCompare(b.method)
})

const collection = {
  version: spec.info?.version ?? '0',
  generatedFrom: 'backend/internal/apidocs/openapi.json',
  tags: (spec.tags ?? []).map((t) => ({ name: t.name, description: firstLine(t.description) })),
  operations,
}

writeFileSync(OUT, JSON.stringify(collection, null, 2) + '\n')

const bytes = readFileSync(OUT).length
console.log(`${operations.length} operasi → public/api-collection.json (${(bytes / 1024).toFixed(1)} KB)`)
