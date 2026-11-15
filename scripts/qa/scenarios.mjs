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

S('README 1 / spec 003: доступ к GitLab')
let p = await path('intern_maria'); out('progress', JSON.stringify(p.progress))
const g = await task('intern_maria', /GitLab/); out('task', g.status, JSON.stringify(g.action))
let ms = await say('intern_maria', 'Подать заявку', { action: g.action, taskId: g.id }); out(sh(ms))
let c = card(ms)
let r = await call('intern_maria', 'POST', `/chat/actions/${c.actionId}/confirm`); out(sh(r.j.messages))
const ext = r.j.messages[0].result?.external; out('external', JSON.stringify(ext))
out('task after confirm', (await task('intern_maria', /GitLab/)).status)
let ap = (await call('lead_artemiy', 'GET', '/approvals')).j; out('approvals', ap.map((a) => `${a.number} ${a.internName} ${a.title} ${a.status}`).join(' | '))
r = await call('lead_artemiy', 'POST', '/approvals/' + ap[0].id, { decision: 'approve' }); out('decide', r.s, r.j.status)
const hist = (await call('intern_maria', 'GET', '/chat/messages')).j.messages.slice(-1); out('chat last', sh(hist))
out('task', (await task('intern_maria', /GitLab/)).status, 'progress', JSON.stringify((await path('intern_maria')).progress))
if (ext?.url) { const b = await fetch(ext.url.replace('8090', '8080/__none')).catch(() => null) }

S('spec 003: запрещенный доступ, трекер, наставник в чате')
out(sh(await say('intern_maria', 'Дай доступ к логам продакшена')))
out(sh(await say('intern_maria', 'Запроси трекер прогресса')))
out('mentor chat', (await call('mentor_igor', 'POST', '/chat/messages', { text: 'привет' })).s, (await call('mentor_igor', 'GET', '/chat/messages')).s)

S('README 2 / spec 001: вопрос о процессах + поиск')
out(sh(await say('intern_maria', 'как устроено код-ревью?')))
for (const q of ['как взять отгул', 'как получить доступ к тестовому стенду']) { const k = (await call('intern_maria', 'GET', '/kb/search?q=' + encodeURIComponent(q))).j; out(q, '->', k.results.slice(0, 3).map((x) => x.title).join(' | ')) }
const kbM = (await call('intern_maria', 'GET', '/kb')).j
out('maria sees Backend', kbM.articles.some((a) => /Кто за что отвечает в команде Backend/.test(a.title)), 'sees Frontend', kbM.articles.some((a) => /Команда Frontend/.test(a.title)))

S('README 2 / spec 004: передача наставнику')
out(sh(ms = await say('intern_maria', 'можно ли работать месяц из другого города?')))
out(sh(await say('intern_maria', 'лучше так: можно ли месяц поработать удаленно из Казани?')))
out(sh(await say('intern_maria', 'да')))
out(sh(await say('intern_maria', 'передай ему вопрос "почему заявка на согласовании"')))
const before = (await call('mentor_igor', 'GET', '/mentor/threads/' + p.intern?.id)).s
const me = (await call('intern_maria', 'GET', '/auth/me')).j; const mid = me.id ?? me.user?.id
let th = (await call('mentor_igor', 'GET', '/mentor/threads/' + mid)).j; out('thread tail', th.messages.slice(-3).map((m) => m.kind + '/' + m.author + ':' + m.text.slice(0, 80)).join(' | '))
r = await call('mentor_igor', 'POST', `/mentor/threads/${mid}/messages`, { text: 'Месяц удаленно можно, согласуй с Артемием за две недели.', addToKb: true }); out('reply', r.s)
out('maria chat', sh((await call('intern_maria', 'GET', '/chat/messages')).j.messages.slice(-1)))
out('lead thread', (await call('lead_artemiy', 'GET', '/mentor/threads/' + mid)).s)
out(sh(await say('intern_maria', 'можно ли работать месяц удаленно из другого города?')))
out(sh(await say('intern_maria', 'как попасть на внутренний хакатон?')))
out(sh(await say('intern_maria', 'нет')))

S('README 3: отгул')
out(sh(ms = await say('intern_maria', 'хочу взять отгул в пятницу')))
c = card(ms); r = await call('intern_maria', 'POST', `/chat/actions/${c.actionId}/confirm`); out(sh(r.j.messages))
ap = (await call('lead_artemiy', 'GET', '/approvals')).j; out('approvals', ap.slice(0, 2).map((a) => `${a.title} ${a.status}`).join(' | '))
r = await call('lead_artemiy', 'POST', '/approvals/' + ap[0].id, { decision: 'approve' }); out('decide', r.s)
out('requests', (await call('intern_maria', 'GET', '/requests')).j.slice(0, 3).map((q) => `${q.number} ${q.title} ${q.status}`).join(' | '))

S('README 4 / spec 002: путь и цели')
p = await path('intern_maria'); out('stages', p.stages.map((s) => s.code + ' ' + s.title + ' ' + s.period).join(' | '))
let gl = (await call('intern_maria', 'GET', '/goals')).j; out('maria goals', gl.goals.map((x) => x.title + ' ' + x.progressPct + '% okr=' + !!x.okr).join(' | '), 'agreed', !!gl.agreed, 'canEdit', gl.canEdit)
const liza = (await call('intern_liza', 'GET', '/auth/me')).j, dima = (await call('intern_dima', 'GET', '/auth/me')).j
out('dima goals', (await call('mentor_igor', 'GET', `/interns/${dima.id}/goals`)).j.goals?.map((x) => x.progressPct).join('/'))
gl = (await call('lead_artemiy', 'GET', `/interns/${liza.id}/goals`)).j; out('liza goals', gl.goals.length, 'agreed', !!gl.agreed, 'canAgree', gl.canAgree)
out('igor agree', (await call('mentor_igor', 'POST', `/interns/${liza.id}/goals/agree`)).s)
r = await call('lead_artemiy', 'POST', `/interns/${liza.id}/goals`, { title: 'Закрыть первую задачу', criterion: 'merge в main', steps: [{ title: 'Получить доступ к GitLab', source: 'task' }, { title: 'Демо команде', source: 'lead' }] }); out('create goal', r.s)
r = await call('lead_artemiy', 'POST', `/interns/${liza.id}/goals/agree`); out('agree', r.s, 'again', (await call('lead_artemiy', 'POST', `/interns/${liza.id}/goals/agree`)).s)
out('maria edit own goal', (await call('intern_maria', 'PATCH', '/goals/' + (await call('intern_maria', 'GET', '/goals')).j.goals[0].id, { title: 'x' })).s)

S('README 5: первая неделя Лизы')
const ch = await task('intern_liza', /чаты/); out('chats task', ch?.status, JSON.stringify(ch?.action))
ms = await say('intern_liza', 'Добавиться в чаты', { action: ch.action, taskId: ch.id }); out(sh(ms))
c = card(ms); r = await call('intern_liza', 'POST', `/chat/actions/${c.actionId}/confirm`); out(sh(r.j.messages))
out('task', (await task('intern_liza', /чаты/)).status)
const from = new Date(Date.now() - 2 * 864e5).toISOString(), to = new Date(Date.now() + 14 * 864e5).toISOString()
const cal = (await call('intern_liza', 'GET', `/me/calendar?from=${from}&to=${to}`)).j; out('calendar', cal.events.map((e) => `${e.kind}/${e.status} ${e.title} ${e.startsAt}`).join(' | '))
const oo = cal.events.find((e) => /1:1/.test(e.title)); r = await call('intern_liza', 'PUT', '/me/notes/' + oo?.id, { text: 'Договорились о синках по средам' }); out('note', r.s, JSON.stringify(r.j))

S('spec 005: заявки в отделы')
out(sh(ms = await say('intern_dima', 'нужен монитор'))); c = card(ms)
if (c) { r = await call('intern_dima', 'POST', `/chat/actions/${c.actionId}/confirm`); out(sh(r.j.messages)) }
out(sh(await say('intern_dima', 'нужен монитор')))
out(sh(await say('intern_dima', 'нужна заявка на оборудование')))
out(sh(await say('intern_dima', 'нужна док-станция')))

S('отзыв заявки')
const reqs = (await call('intern_dima', 'GET', '/requests')).j; out('dima reqs', reqs.slice(0, 3).map((q) => `${q.number} ${q.status} canCancel=${q.canCancel}`).join(' | '))
const w = reqs.find((q) => q.status === 'waiting'); if (w) { out(sh(await say('intern_dima', `отзови заявку №${w.number}`))); out(sh(await say('intern_dima', 'да'))) }
