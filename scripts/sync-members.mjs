/**
 * Sync members & chapters dari BNI VM ke Supabase.
 *
 * Secrets are read from env — NEVER hardcode them (this file is committed):
 *   BNI_VM_TOKEN=...  SUPABASE_URL=...  SUPABASE_SERVICE_KEY=...  node scripts/sync-members.mjs
 */

const BNI_VM_URL   = process.env.BNI_VM_URL || 'https://www.bni-vh.com/api/external/v1'
const BNI_VM_TOKEN = process.env.BNI_VM_TOKEN
const SUPABASE_URL = process.env.SUPABASE_URL
const SERVICE_KEY  = process.env.SUPABASE_SERVICE_KEY

if (!BNI_VM_TOKEN || !SUPABASE_URL || !SERVICE_KEY) {
  console.error('Missing env: set BNI_VM_TOKEN, SUPABASE_URL, and SUPABASE_SERVICE_KEY before running.')
  process.exit(1)
}

const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${SERVICE_KEY}`,
  'apikey': SERVICE_KEY,
  'Prefer': 'resolution=merge-duplicates',
}

async function fetchAllMembers() {
  let all = []
  let offset = 0
  while (true) {
    const r = await fetch(`${BNI_VM_URL}/members?limit=200&offset=${offset}`, {
      headers: { Authorization: `Bearer ${BNI_VM_TOKEN}` },
    })
    if (!r.ok) throw new Error(`BNI VM error: ${r.status} ${await r.text()}`)
    const j = await r.json()
    all = all.concat(j.data)
    console.log(`  Fetched ${all.length}/${j.pagination.total} members...`)
    if (!j.pagination.hasMore) break
    offset += 200
  }
  return all
}

async function upsert(table, rows) {
  const r = await fetch(`${SUPABASE_URL}/rest/v1/${table}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(rows),
  })
  const text = await r.text()
  if (r.status !== 201 && r.status !== 200) {
    throw new Error(`Supabase ${table} error: ${r.status} ${text}`)
  }
}

async function main() {
  console.log('📥 Fetching members dari BNI VM...')
  const members = await fetchAllMembers()

  const now = new Date().toISOString()

  // Upsert chapters
  const chaptersMap = {}
  for (const m of members) {
    if (!chaptersMap[m.chapter_id]) chaptersMap[m.chapter_id] = m.chapter
  }
  const chapterRows = Object.entries(chaptersMap).map(([id, name]) => ({
    id, name, display_name: name, synced_at: now,
  }))
  console.log(`\n📂 Upserting ${chapterRows.length} chapters...`)
  await upsert('chapters', chapterRows)
  chapterRows.forEach(c => console.log(`  ✓ ${c.name}`))

  // Upsert members
  const memberRows = members.map(m => ({
    id:             m.id,
    chapter_id:     m.chapter_id,
    name:           m.name,
    email:          m.email    || null,
    phone:          m.phone    || null,
    company:        m.company  || null,
    business_field: m.business_field || null,
    status:         m.status   || 'active',
    joined_date:    m.joined_date,
    synced_at:      now,
  }))
  console.log(`\n👥 Upserting ${memberRows.length} members...`)
  await upsert('members', memberRows)

  // Preview 5 sample
  console.log('\nSample data:')
  memberRows.slice(0, 5).forEach(m =>
    console.log(`  • ${m.name} | ${m.company ?? '-'} | ${m.business_field ?? '-'}`)
  )

  console.log(`\n✅ Selesai — ${memberRows.length} members & ${chapterRows.length} chapters berhasil di-sync.`)
}

main().catch(err => { console.error('❌', err.message); process.exit(1) })
