import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { describe, expect, it } from 'vitest'
import { API_URL } from '../src/api/client'
import { shortDate } from '../src/lib/dates'
import { buildTodos, daysLate, plural } from '../src/lib/mentorHome'
import { buildMockPath } from '../src/mocks/pathData'
import { server } from '../src/mocks/server'
import { loginAs, renderApp } from './test-utils'

const T = { timeout: 4000 }
const NOW = new Date('2026-11-12T12:00:00+03:00')

describe('расчеты главной наставника', () => {
  it('просрочка, согласование, вопросы и цели', () => {
    expect(daysLate(null, NOW)).toBe(0)
    expect(daysLate('2026-11-12', NOW)).toBe(0)
    expect(daysLate('2026-11-10', NOW)).toBe(2)
    expect(plural(1, 'вопрос', 'вопроса', 'вопросов')).toBe('вопрос')
    expect(plural(3, 'вопрос', 'вопроса', 'вопросов')).toBe('вопроса')
    expect(plural(12, 'вопрос', 'вопроса', 'вопросов')).toBe('вопросов')
    const todos = buildTodos(
      {
        internId: 'u-anna',
        internName: 'Мария',
        stageCode: '02',
        stageTitle: 'Доступы',
        lastText: 'Какая роль нужна?',
        lastAt: NOW.toISOString(),
        unread: 2,
      },
      buildMockPath(),
      {
        internId: 'u-anna',
        agreed: null,
        fromOkr: false,
        goals: [],
        okrs: [],
        canEdit: true,
        canAgree: false,
      },
      NOW,
    )
    const kinds = todos.map((t) => t.kind)
    expect(kinds).toEqual(expect.arrayContaining(['question', 'late', 'waiting', 'goals']))
    expect(todos[0].chip).toBe('2 вопроса')
    expect(buildTodos(undefined, undefined, undefined, NOW)).toEqual([])
  })

  it('короткая дата', () => {
    expect(shortDate('2026-11-10')).toBe('10 нояб')
    expect(shortDate(null)).toBe('без срока')
  })
})

describe('главная наставника', () => {
  it('стажеры с метками, раскрытие и смена статуса задачи', async () => {
    loginAs('mentor_igor')
    renderApp('/')
    expect(await screen.findByRole('heading', { name: 'Привет, Игорь!' }, T)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Мои стажеры' })).toBeInTheDocument()
    const anna = await screen.findByRole('button', { name: /^Мария:.*вопрос/ }, T)
    expect(anna).toHaveAttribute('aria-expanded', 'true')
    const panel = await screen.findByRole('region', { name: 'Мария: подробности' }, T)
    expect(within(panel).getAllByRole('link', { name: 'Ответить' })[0]).toHaveAttribute(
      'href',
      '/inbox?intern=u-anna',
    )
    const status = await within(panel).findByRole(
      'button',
      { name: /Статус «Получить доступ к GitLab»: в работе/ },
      T,
    )
    await userEvent.click(status)
    expect(
      await within(panel).findByRole(
        'button',
        { name: /Статус «Получить доступ к GitLab»: готово/ },
        T,
      ),
    ).toBeInTheDocument()
    // Павел: цели не согласованы
    const pavel = screen.getByRole('button', { name: /^Павел:/ })
    await userEvent.click(pavel)
    expect(await screen.findByRole('region', { name: 'Павел: подробности' }, T)).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Мария: подробности' })).not.toBeInTheDocument()
    expect(await screen.findByText('Цели еще не согласованы', {}, T)).toBeInTheDocument()
  })
})

describe('путь стажера: подробности задачи', () => {
  it('клик по задаче раскрывает описание, срок и цель', async () => {
    loginAs('intern_maria')
    renderApp('/?stage=02')
    const task = await screen.findByRole(
      'button',
      { name: 'Получить доступ к GitLab: подробнее' },
      T,
    )
    expect(task).toHaveAttribute('aria-expanded', 'false')
    await userEvent.click(task)
    expect(task).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/Нужна группа backend с правами Developer/)).toBeInTheDocument()
    expect(screen.getByText('10 нояб')).toBeInTheDocument()
    expect(screen.getByText('Пройти путь задачи до релиза', { selector: 'dd' })).toBeInTheDocument()
    await userEvent.click(task)
    expect(screen.queryByText(/Нужна группа backend/)).not.toBeInTheDocument()
  })
})

describe('справочник наставника', () => {
  it('только просмотр: без «Спросить AI-buddy»', async () => {
    loginAs('mentor_igor')
    renderApp('/kb')
    expect(await screen.findByText('Только для просмотра', {}, T)).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Спросить AI-buddy' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Перейти в «Материалы»' })).toHaveAttribute(
      'href',
      '/materials',
    )
  })
})

describe('руководитель: цели стажеров', () => {
  it('правит черновик Павла и согласует цели', async () => {
    loginAs('lead_artemiy')
    renderApp('/')
    await userEvent.click(await screen.findByRole('button', { name: /Павел/ }, T))
    const panel = await screen.findByRole('region', { name: 'Цели: Павел' }, T)
    expect(within(panel).getByText('еще не согласованы с руководителем')).toBeInTheDocument()

    // новая цель: без шагов не сохраняется
    await userEvent.click(within(panel).getByRole('button', { name: /Цель/ }))
    const dialog = await screen.findByRole('dialog', {}, T)
    await userEvent.type(within(dialog).getByLabelText('Цель'), 'Провести демо')
    await userEvent.type(within(dialog).getByLabelText('Критерий успеха'), 'демо на 8 неделе')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(within(dialog).getByRole('alert')).toHaveTextContent('хотя бы один шаг')
    await userEvent.type(within(dialog).getByLabelText('Шаг 1'), 'Подготовить слайды')
    await userEvent.selectOptions(within(dialog).getByLabelText('Кто закрывает шаг 2'), 'lead')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Убрать шаг 2' }))
    await userEvent.click(within(dialog).getByRole('button', { name: '+ Шаг' }))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(await within(panel).findByText('Провести демо', {}, T)).toBeInTheDocument()

    // изменить и удалить
    await userEvent.click(
      within(panel).getByRole('button', { name: 'Изменить цель «Провести демо»' }),
    )
    const edit = await screen.findByRole('dialog', {}, T)
    const title = within(edit).getByLabelText('Цель')
    await userEvent.clear(title)
    await userEvent.type(title, 'Провести демо команде')
    await userEvent.click(within(edit).getByRole('button', { name: 'Сохранить' }))
    expect(await within(panel).findByText('Провести демо команде', {}, T)).toBeInTheDocument()
    await userEvent.click(
      within(panel).getByRole('button', { name: 'Удалить цель «Провести демо команде»' }),
    )
    await waitFor(
      () => expect(within(panel).queryByText('Провести демо команде')).not.toBeInTheDocument(),
      T,
    )

    await userEvent.click(within(panel).getByRole('button', { name: 'Согласовать цели' }))
    expect(await within(panel).findByText(/согласованы с Артемием/, {}, T)).toBeInTheDocument()
    expect(
      within(panel).queryByRole('button', { name: 'Согласовать цели' }),
    ).not.toBeInTheDocument()
  })

  it('подтверждает свой шаг в согласованных целях Марии', async () => {
    loginAs('lead_artemiy')
    renderApp('/')
    const panel = await screen.findByRole('region', { name: 'Цели: Мария' }, T)
    await userEvent.click(await within(panel).findByRole('button', { name: 'Подтвердить' }, T))
    await waitFor(
      () =>
        expect(
          within(panel).queryByRole('button', { name: 'Подтвердить' }),
        ).not.toBeInTheDocument(),
      T,
    )
    expect(within(panel).getByText('50%')).toBeInTheDocument()
  })

  it('ошибка списка стажеров и пустой список', async () => {
    loginAs('lead_artemiy')
    server.use(http.get(`${API_URL}/api/interns`, () => HttpResponse.json([])))
    renderApp('/')
    expect(await screen.findByText('В команде пока нет стажеров.', {}, T)).toBeInTheDocument()
  })
})

describe('руководитель: согласования', () => {
  it('согласует заявку и отклоняет только с причиной', async () => {
    loginAs('lead_artemiy')
    renderApp('/approvals')
    const gitlab = await screen.findByRole('region', { name: /№1042/ }, T)
    await userEvent.click(within(gitlab).getByRole('button', { name: 'Согласовать' }))
    expect(await within(gitlab).findByText('согласовано', {}, T)).toBeInTheDocument()

    const stand = screen.getByRole('region', { name: /№1039/ })
    await userEvent.click(within(stand).getByRole('button', { name: 'Отклонить' }))
    expect(within(stand).getByText('Напиши причину: ее увидит стажер')).toBeInTheDocument()
    await userEvent.type(
      within(stand).getByLabelText('Причина отказа'),
      'Стенд не нужен на этом этапе',
    )
    await userEvent.click(within(stand).getByRole('button', { name: 'Отклонить' }))
    expect(await within(stand).findByText('отклонено', {}, T)).toBeInTheDocument()
    expect(within(stand).getByText('Стенд не нужен на этом этапе')).toBeInTheDocument()
    expect(await screen.findByText(/Все заявки разобраны/, {}, T)).toBeInTheDocument()
  })

  it('наставника на согласования не пускает', async () => {
    loginAs('mentor_igor')
    renderApp('/approvals')
    expect(await screen.findByRole('heading', { name: 'Привет, Игорь!' }, T)).toBeInTheDocument()
  })
})

describe('правки после проверки на стенде', () => {
  it('настройки у каждого аккаунта свои', async () => {
    const { useSettingsStore } = await import('../src/store/settingsStore')
    loginAs('intern_maria')
    await waitFor(() => expect(useSettingsStore.getState().answerStyle).toBe('normal'))
    useSettingsStore.getState().setAnswerStyle('detailed')
    loginAs('mentor_igor')
    await waitFor(() => expect(useSettingsStore.getState().answerStyle).toBe('normal'))
    useSettingsStore.getState().setAnswerStyle('short')
    loginAs('intern_maria')
    await waitFor(() => expect(useSettingsStore.getState().answerStyle).toBe('detailed'))
    loginAs('mentor_igor')
    await waitFor(() => expect(useSettingsStore.getState().answerStyle).toBe('short'))
  })

  it('руководитель видит поддержку по техническим сбоям', async () => {
    loginAs('lead_artemiy')
    renderApp('/support')
    expect(await screen.findByText('не работает?', {}, T)).toBeInTheDocument()
    expect(screen.getAllByText('Не сохраняются цели стажера').length).toBeGreaterThan(0)
    expect(screen.getByRole('link', { name: 'Поддержка' })).toBeInTheDocument()
    expect(screen.queryByLabelText(/Приложить диалог/)).not.toBeInTheDocument()
  })

  it('наставник видит пользу AI-buddy одной строкой, подробности по клику', async () => {
    loginAs('mentor_igor')
    renderApp('/')
    const line = await screen.findByRole('button', { name: 'Польза AI-buddy за неделю' }, T)
    expect(line).toHaveTextContent('закрыл 14 вопросов без тебя, это около 2,8 ч')
    await userEvent.click(line)
    expect(await screen.findByText('14 из 17')).toBeInTheDocument()
    expect(screen.getByText('в среднем 6 ч')).toBeInTheDocument()
    expect(screen.getByText('82%')).toBeInTheDocument()
  })

  it('куратор стажировки – в карточке HR, кнопки контактов работают', async () => {
    loginAs('intern_maria')
    const router = renderApp('/support')
    expect(await screen.findByText(/куратор стажировки \(HR BP\)/, {}, T)).toBeInTheDocument()
    expect(screen.getByText(/Вопросы к руководителю обсуди с наставником/)).toBeInTheDocument()
    await userEvent.click(screen.getAllByRole('button', { name: 'Оформить заявку' })[0]!)
    expect(router.state.location.pathname).toBe('/chat')
    await router.navigate({ to: '/support' })
    await userEvent.click(await screen.findByRole('button', { name: 'Написать' }, T))
    expect(router.state.location.pathname).toBe('/mentor')
  })

  it('у наставника кнопки контактов ведут к форме обращения', async () => {
    loginAs('mentor_igor')
    renderApp('/support')
    const buttons = await screen.findAllByRole('button', { name: 'Сообщить о проблеме' }, T)
    await userEvent.click(buttons[0]!)
    expect(screen.getByLabelText('Подробности')).toHaveFocus()
  })

  it('null в external значит «не в Jira»: показывается обычный номер', async () => {
    const { jiraKey, requestRef } = await import('../src/lib/jira')
    expect(jiraKey({ kind: 'access', number: '7', external: null })).toBeNull()
    expect(requestRef({ kind: 'access', number: '7', external: null })).toBe('№7')
    expect(requestRef({ kind: 'access', number: '7' })).toBe('ACCESS-7')
    expect(
      requestRef({
        kind: 'access',
        number: '7',
        external: { system: 'J', key: 'ACCESS-8001', url: null },
      }),
    ).toBe('ACCESS-8001')
  })
})
