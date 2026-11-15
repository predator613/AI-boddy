const A = 'http://localhost:8080/api', T = {}
const out = (...x) => console.log(...x)
async function call(u, m, p, b, raw) {
  const r = await fetch(A + p, { method: m, headers: { 'Content-Type': 'application/json', ...(u ? { Authorization: 'Bearer ' + T[u] } : {}) }, body: b ? JSON.stringify(b) : undefined })
  let j = null; try { j = await r.json() } catch {}
  return { s: r.status, j }
}
const login = async (u) => (T[u] = (await call(null, 'POST', '/auth/login', { login: u, password: 'demo123' })).j.token)
const say = async (u, text, extra = {}) => { const r = await call(u, 'POST', '/chat/messages', { text, ...extra }); return r.j?.messages ?? [{ kind: 'HTTP' + r.s, text: JSON.stringify(r.j) }] }
const sh = (ms) => ms.map((m) => `   [${m.kind}${m.confirm ? ' ' + m.confirm.state : ''}] ${m.text.slice(0, 220).replace(/\n/g, ' ')}${m.confirm ? ' ' + JSON.stringify(m.confirm.fields.map((f) => f.label + '=' + f.value)) : ''}${m.result ? ' RES ' + JSON.stringify(m.result) : ''}`).join('\n')
const card = (ms) => ms.find((m) => m.confirm?.state === 'pending')?.confirm
const S = (n) => out('\n===== ' + n)
const wait = (ms) => new Promise((r) => setTimeout(r, ms))
const path = async (u) => (await call(u, 'GET', '/path')).j
const task = async (u, re) => (await path(u)).stages.flatMap((s) => s.tasks).find((t) => re.test(t.title))

for (const u of ['intern_maria', 'intern_liza', 'intern_dima', 'mentor_igor', 'lead_artemiy']) await login(u)

let r

const st = async (u, m, p, b) => (await call(u, m, p, b)).s
const id = async (u) => (await call(u, 'GET', '/auth/me')).j.id
const [maria, dima, liza] = [await id('intern_maria'), await id('intern_dima'), await id('intern_liza')]
S('auth')
out('no token', await st(null, 'GET', '/path'), 'bad token', (await fetch(A + '/path', { headers: { Authorization: 'Bearer x.y.z' } })).status)
const [h, pl, sg] = T.intern_maria.split('.'); const pj = JSON.parse(Buffer.from(pl, 'base64url')); pj.role = 'lead'; pj.sub = pj.sub
const forged = h + '.' + Buffer.from(JSON.stringify(pj)).toString('base64url') + '.' + sg
out('forged payload', (await fetch(A + '/approvals', { headers: { Authorization: 'Bearer ' + forged } })).status, 'payload', JSON.stringify(pj))
const alg = Buffer.from(JSON.stringify({ alg: 'none', typ: 'JWT' })).toString('base64url') + '.' + pl + '.'
out('alg none', (await fetch(A + '/path', { headers: { Authorization: 'Bearer ' + alg } })).status)
out('wrong pwd', await st(null, 'POST', '/auth/login', { login: 'intern_maria', password: 'x' }), 'sql login', await st(null, 'POST', '/auth/login', { login: "' OR 1=1 --", password: 'x' }))
S('IDOR')
const mreq = (await call('intern_maria', 'GET', '/requests')).j[0]
out('dima cancel maria req', await st('intern_dima', 'POST', `/requests/${mreq.id}/cancel`), await st('intern_dima', 'DELETE', `/requests/${mreq.id}`))
let ms = await say('intern_maria', 'хочу отгул 27 ноября'); const c = card(ms)
out('dima confirm maria draft', await st('intern_dima', 'POST', `/chat/actions/${c.actionId}/confirm`), 'cancel', await st('intern_dima', 'POST', `/chat/actions/${c.actionId}/cancel`))
out('igor approve maria', await st('mentor_igor', 'POST', `/approvals/${c.actionId}`, { decision: 'approve' }))
out('dima path of maria', await st('intern_dima', 'GET', `/interns/${maria}/path`), 'dima goals maria', await st('intern_dima', 'GET', `/interns/${maria}/goals`))
out('igor path maria', await st('mentor_igor', 'GET', `/interns/${maria}/path`), 'igor checklist maria', await st('mentor_igor', 'GET', `/mentor/interns/${maria}/checklist`))
const ev = (await call('intern_maria', 'GET', `/me/calendar?from=${new Date(Date.now() - 864e5).toISOString()}&to=${new Date(Date.now() + 14 * 864e5).toISOString()}`)).j.events
const own = ev.find((e) => e.status === 'own')
out('dima note on maria event', await st('intern_dima', 'PUT', `/me/notes/${ev[0].id}`, { text: 'x' }), 'dima delete maria entry', own ? await st('intern_dima', 'DELETE', `/me/calendar/entries/${own.id}`) : 'n/a')
const kbM = (await call('intern_maria', 'GET', '/kb')).j; const bArt = kbM.articles.find((a) => /Backend/.test(a.title))
out('dima read backend article', await st('intern_dima', 'GET', `/kb/articles/${bArt.id}`))
out('intern console', await st('intern_maria', 'GET', '/console/templates'), await st('intern_maria', 'POST', '/console/stages', { title: 'x' }))
out('intern materials post/delete', await st('intern_maria', 'DELETE', `/materials/00000000-0000-4000-8000-000000000000`))
out('dialogs of others', await st('intern_dima', 'GET', `/chat/dialogs/${(await call('intern_maria', 'GET', '/chat/dialogs')).j[0]?.id}/messages`))
S('injection/XSS')
out('kb search sql', (await call('intern_maria', 'GET', '/kb/search?q=' + encodeURIComponent("'; DROP TABLE users; --"))).s, 'users ok', await st('intern_maria', 'GET', '/auth/me'))
const x = '<img src=x onerror=alert(1)>'
ms = await say('intern_maria', x); out('chat xss echo', JSON.stringify(ms.map((m) => m.text)).slice(0, 200))
await call('mentor_igor', 'POST', `/mentor/threads/${maria}/messages`, { text: x })
const fd = new FormData(); fd.append('title', x); fd.append('url', 'javascript:alert(1)')
out('material js url', (await fetch(A + '/materials', { method: 'POST', headers: { Authorization: 'Bearer ' + T.mentor_igor }, body: fd })).status)
const fd2 = new FormData(); fd2.append('title', 'x'); fd2.append('file', new Blob(['<script>alert(1)</script>']), 'a.html')
out('material html file', (await fetch(A + '/materials', { method: 'POST', headers: { Authorization: 'Bearer ' + T.mentor_igor }, body: fd2 })).status)
r = await call('intern_maria', 'POST', '/support/tickets', { type: x, details: x }); out('ticket xss stored', r.s, r.j?.number)
S('limits/CORS')
out('huge chat', await st('intern_maria', 'POST', '/chat/messages', { text: 'a'.repeat(5000) }))
out('bad json', (await fetch(A + '/chat/messages', { method: 'POST', headers: { Authorization: 'Bearer ' + T.intern_maria, 'Content-Type': 'application/json' }, body: '{bad' })).status)
const cors = await fetch(A + '/auth/test-users', { headers: { Origin: 'https://evil.example' } }); out('cors evil', cors.headers.get('access-control-allow-origin'))
const pre = await fetch(A + '/path', { method: 'OPTIONS', headers: { Origin: 'https://evil.example', 'Access-Control-Request-Method': 'GET' } }); out('preflight evil', pre.status, pre.headers.get('access-control-allow-origin'))
let rl = []; for (let i = 0; i < 25; i++) rl.push((await call(null, 'POST', '/auth/login', { login: 'intern_maria', password: 'bad' })).s); out('25 bad logins', [...new Set(rl)].join(','))
out('security headers', JSON.stringify(Object.fromEntries([...(await fetch(A + '/health')).headers].filter(([k]) => /x-|content-security|strict/.test(k)))))
