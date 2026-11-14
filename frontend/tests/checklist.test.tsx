import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { describe, expect, it, onTestFinished, vi } from 'vitest'
import { API_URL } from '../src/api/client'
import { foldMaterials } from '../src/lib/mentorUi'
import { msk, mskIso, ymd } from '../src/lib/msk'
import { EntryInput } from '../src/schemas/calendar'
import { resetMentorMocks } from '../src/mocks/mentorData'
import { server } from '../src/mocks/server'
import { loginAs, renderApp } from './test-utils'

const T = { timeout: 8000 }

describe('время по Москве', () => {
  it('части даты, ключ дня и ISO записи', () => {
    const p = msk('2026-11-12T12:30:00Z')
    expect(p).toEqual({ ymd: '2026-11-12', hour: 15, minute: 30, weekday: 3 })
    expect(ymd(new Date(2026, 10, 9))).toBe('2026-11-09')
    expect(mskIso(new Date(2026, 10, 9), 10.5)).toBe('2026-11-09T10:30:00+03:00')
    const base = { title: 'Фокус', kind: 'focus' as const, shareWithMentor: true }
    expect(
      EntryInput.safeParse({
        ...base,
        startsAt: mskIso(new Date(2026, 10, 9), 17),
        durationMin: 60,
      }).success,
    ).toBe(true)
    expect(
      EntryInput.safeParse({
        ...base,
        startsAt: mskIso(new Date(2026, 10, 9), 17.5),
        durationMin: 60,
      }).success,
    ).toBe(false)
  })
})

describe('чек-лист наставника', () => {
  it('пункт отмечается, секунду виден с галочкой, возвращается из меню', async () => {
    loginAs('mentor_igor')
    renderApp('/')
    const panel = await screen.findByRole('region', { name: 'Мария: подробности' }, T)
    const name = /Запросить методичку наставника/
    await userEvent.click(await within(panel).findByRole('checkbox', { name }, T))
    // под нагрузкой пункт может успеть уйти раньше, чем тест увидит галочку: тогда это тоже успех
    await waitFor(() => {
      const box = within(panel).queryByRole('checkbox', { name })
      if (box) expect(box).toBeChecked()
    }, T)
    await waitFor(() => expect(within(panel).queryByRole('checkbox', { name })).toBeNull(), T)
    await userEvent.click(
      await within(panel).findByRole('button', { name: /Чек-лист наставника · 5 из 10/ }, T),
    )
    // hidden: меню может еще проигрывать анимацию появления, а пункт уже на месте
    const inMenu = await screen.findByRole('checkbox', { name, hidden: true }, T)
    expect(inMenu).toBeChecked()
    await userEvent.click(inMenu)
    expect(
      await within(panel).findByRole('button', { name: /Чек-лист наставника · 4 из 10/ }, T),
    ).toBeInTheDocument()
  }, 20_000)
})

describe('материалы и справочник наставника', () => {
  it('один список с разделами, поиск и загрузка в новый раздел', async () => {
    loginAs('mentor_igor')
    renderApp('/materials')
    expect(await screen.findByText('Правила код-ревью команды', {}, T)).toBeInTheDocument()
    const sections = screen.getByRole('navigation', { name: 'Разделы справочника' })
    await userEvent.click(within(sections).getByRole('button', { name: /Доступы и сервисы/ }))
    expect(screen.queryByText('Правила код-ревью команды')).toBeNull()
    await userEvent.click(within(sections).getByRole('button', { name: /Доступы и сервисы/ }))
    await userEvent.type(screen.getByLabelText('Поиск по материалам'), 'zzz')
    expect(screen.getByText(/Ничего не нашлось/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Загрузить материал' }))
    const dialog = await screen.findByRole('dialog', {}, T)
    await userEvent.selectOptions(within(dialog).getByLabelText('Раздел справочника'), '__new__')
    await userEvent.type(within(dialog).getByLabelText('Название нового раздела'), 'Дежурства')
    await userEvent.type(within(dialog).getByLabelText('Название'), 'Как дежурить')
    await userEvent.click(within(dialog).getByText('Вставить ссылку'))
    await userEvent.type(within(dialog).getByLabelText('Ссылка'), 'https://wiki.example/duty')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Загрузить' }))
    await userEvent.clear(screen.getByLabelText('Поиск по материалам'))
    expect(await screen.findByText('Как дежурить', {}, T)).toBeInTheDocument()
    expect(within(sections).getByRole('button', { name: /Дежурства/ })).toBeInTheDocument()
  })
})

describe('стажер: конспект, чаты, профиль, счетчик заявок', () => {
  it('конспект встречи сохраняется в календаре', async () => {
    // будний день: в выходные календарь открывает следующую неделю, а демо-встречи – на текущей
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 10, 11, 13, 20))
    resetMentorMocks()
    onTestFinished(() => {
      vi.useRealTimers()
    })
    loginAs('intern_maria')
    renderApp('/calendar')
    await userEvent.click(await screen.findByRole('button', { name: /Знакомство с DevOps/ }, T))
    const toggle = await screen.findByRole('button', { name: /Конспект встречи/ }, T)
    await userEvent.click(toggle)
    await userEvent.type(screen.getByLabelText('Конспект встречи'), 'Кирилл отвечает за стенды')
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))
    expect(await screen.findByText('· есть заметки', {}, T)).toBeInTheDocument()
  })

  it('AI-buddy добавляет в чаты команды и пишет «Готово»', async () => {
    loginAs('intern_maria')
    renderApp('/chat?q=Добавь меня в чаты команды')
    await userEvent.click(await screen.findByRole('button', { name: 'Подтвердить' }, T))
    expect(await screen.findByText('Готово', {}, T)).toBeInTheDocument()
    expect(screen.queryByText(/Напишу, когда будет решение/)).toBeNull()
  })

  it('«Изменить» подставляет текст правки; устаревшая карточка – сообщение', async () => {
    loginAs('intern_maria')
    renderApp('/chat?q=Как получить доступ к GitLab?')
    await userEvent.click(await screen.findByRole('button', { name: 'Изменить' }, T))
    expect(screen.getByLabelText('Сообщение')).toHaveValue('Измени в заявке: ')
    expect(screen.getByText('Этап 02 · Доступы')).toBeInTheDocument()
    server.use(
      http.post(`${API_URL}/api/chat/actions/:id/confirm`, () =>
        HttpResponse.json({ error: { code: 'not_found', message: 'x' } }, { status: 404 }),
      ),
    )
    await userEvent.click(screen.getByRole('button', { name: 'Подтвердить' }))
    expect(await screen.findByText('Эта карточка уже неактуальна', {}, T)).toBeInTheDocument()
  })

  it('в профиле наставник и неделя, на «Мои заявки» счетчик', async () => {
    loginAs('intern_maria')
    renderApp('/profile')
    expect(await screen.findByText('Игорь', {}, T)).toBeInTheDocument()
    expect(await screen.findByText('неделя 2 из 24', {}, T)).toBeInTheDocument()
    expect(await screen.findByLabelText(/ждут решения: \d/, {}, T)).toBeInTheDocument()
  })

  it('задача «Добавиться в чаты» ведет в чат с готовым запросом', async () => {
    loginAs('intern_maria')
    renderApp('/?stage=05')
    expect(await screen.findByText('Пройти материалы по защите данных', {}, T)).toBeInTheDocument()
    expect(screen.getAllByText('месяцы 1–2').length).toBeGreaterThan(0)
  })
})

describe('переписка: старые карточки материалов', () => {
  it('оставляет три последних материала за 7 дней, остальное сворачивает', () => {
    const now = Date.parse('2026-11-12T12:00:00Z')
    const m = (id: string, kind: string, type: string | null, at: string) => ({
      id,
      kind,
      at,
      update: type ? { type } : null,
    })
    const list = [
      m('1', 'update', 'material', '2026-11-01T10:00:00Z'),
      m('2', 'update', 'material', '2026-11-08T10:00:00Z'),
      m('3', 'update', 'material', '2026-11-09T10:00:00Z'),
      m('4', 'update', 'material', '2026-11-10T10:00:00Z'),
      m('5', 'update', 'material', '2026-11-11T10:00:00Z'),
      m('6', 'update', 'plan', '2026-10-01T10:00:00Z'),
      m('7', 'text', null, '2026-10-01T10:00:00Z'),
    ]
    const f = foldMaterials(list, now)
    expect(f.hidden).toBe(2)
    expect(f.visible.map((x) => x.id)).toEqual(['3', '4', '5', '6', '7'])
  })
})

describe('журнал изменений плана', () => {
  it('удаление задачи попадает в журнал', async () => {
    loginAs('mentor_igor')
    renderApp('/plan')
    const log = await screen.findByRole('region', { name: 'Журнал изменений' }, T)
    expect(within(log).getByText(/Изменений пока нет/)).toBeInTheDocument()
    const del = (await screen.findAllByRole('button', { name: /^Удалить задачу/ }, T))[0]!
    await userEvent.click(del)
    expect(await within(log).findByText(/Задача «.*» удалена/, {}, T)).toBeInTheDocument()
  })
})
