import { cleanup, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { API_URL } from '../src/api/client'
import { server } from '../src/mocks/server'
import { addDays, mondayOf, tasksWord, timeLabel } from '../src/lib/mentorUi'
import { loginAs, renderApp } from './test-utils'

const T = { timeout: 3000 }
const toastText = async (re: RegExp) =>
  within(await screen.findByRole('status', {}, T)).findByText(re, {}, T)

describe('меню по ролям', () => {
  it('наставник видит кабинет наставника, а не заявки', async () => {
    loginAs('mentor_igor')
    renderApp('/')
    await screen.findByRole('heading', { name: 'Мои стажеры' }, T)
    for (const name of ['Сообщения', 'План адаптации', 'Материалы и справочник', 'Расписание'])
      expect(screen.getByRole('link', { name })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Мои заявки' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Чат с наставником' })).not.toBeInTheDocument()
  })

  it('стажера не пускает в разделы наставника, а наставника – в чат стажера', async () => {
    loginAs('intern_anna')
    const router = renderApp('/plan')
    await screen.findByRole('heading', { name: 'Привет, Анна!' }, T)
    expect(router.state.location.pathname).toBe('/')
    expect(screen.getByRole('link', { name: 'Чат с наставником' })).toBeInTheDocument()
  })

  it('наставника с /mentor отправляет на главную', async () => {
    loginAs('mentor_igor')
    const router = renderApp('/mentor')
    await screen.findByRole('heading', { name: 'Мои стажеры' }, T)
    expect(router.state.location.pathname).toBe('/')
  })
})

describe('колокольчик наставника', () => {
  it('показывает, что изменилось, и отмечает прочитанным', async () => {
    loginAs('mentor_igor')
    renderApp('/')
    await userEvent.click(await screen.findByRole('button', { name: 'Уведомления, новых: 3' }, T))
    expect(await screen.findByText('Олег согласовал цели Анны', {}, T)).toBeInTheDocument()
    await userEvent.click(await screen.findByText('Прочитать все', {}, T))
    expect(
      await screen.findByRole('button', { name: 'Уведомления, новых: 0' }, T),
    ).toBeInTheDocument()
  })
})

describe('чат стажера с наставником', () => {
  it('показывает переписку, передачу от AI-buddy и карточки изменений', async () => {
    loginAs('intern_anna')
    renderApp('/mentor')
    expect(await screen.findByRole('heading', { name: 'Игорь' }, T)).toBeInTheDocument()
    expect(screen.getByText(/AI-buddy передал Игорю вопрос про роль в GitLab/)).toBeInTheDocument()
    expect(screen.getByText('Правила код-ревью команды')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Отметить прочитанным' }))
    expect(await screen.findByText('Прочитано', {}, T)).toBeInTheDocument()
    expect(await toastText(/наставник видит ответ/)).toBeInTheDocument()
  })

  it('фильтр «только изменения» скрывает обычные сообщения', async () => {
    loginAs('intern_anna')
    renderApp('/mentor')
    await screen.findByText('Игорь, для GitLab нужна роль Developer или Maintainer?', {}, T)
    await userEvent.click(screen.getByLabelText('Только изменения от наставника'))
    expect(
      screen.queryByText('Игорь, для GitLab нужна роль Developer или Maintainer?'),
    ).not.toBeInTheDocument()
    expect(screen.getByText('Правила код-ревью команды')).toBeInTheDocument()
  })

  it('сообщение стажера уходит наставнику во «Входящие»', async () => {
    loginAs('intern_anna')
    renderApp('/mentor')
    const box = await screen.findByLabelText('Сообщение наставнику', {}, T)
    await userEvent.click(screen.getByRole('button', { name: 'Спасибо!' }))
    expect(box).toHaveValue('Спасибо!')
    await userEvent.clear(box)
    await userEvent.type(box, 'Посмотришь мой MR?{Enter}')
    expect(await screen.findByText('Посмотришь мой MR?', {}, T)).toBeInTheDocument()
    expect(await toastText(/Игорь получит уведомление/)).toBeInTheDocument()
    expect(box).toHaveValue('')
  })

  it('ошибка отправки видна, текст не теряется', async () => {
    server.use(
      http.post(`${API_URL}/api/me/mentor-thread/messages`, () =>
        HttpResponse.json({ error: { code: 'internal_error', message: 'x' } }, { status: 500 }),
      ),
    )
    loginAs('intern_anna')
    renderApp('/mentor')
    const box = await screen.findByLabelText('Сообщение наставнику', {}, T)
    await userEvent.type(box, 'Привет{Enter}')
    expect(await screen.findByRole('alert', {}, T)).toBeInTheDocument()
    expect(box).toHaveValue('Привет')
  })

  it('ошибка загрузки переписки с повтором', async () => {
    server.use(
      http.get(`${API_URL}/api/me/mentor-thread`, () =>
        HttpResponse.json({ error: { code: 'internal_error', message: 'x' } }, { status: 500 }),
      ),
    )
    loginAs('intern_anna')
    renderApp('/mentor')
    expect(await screen.findByRole('button', { name: 'Повторить' }, T)).toBeInTheDocument()
  })
})

describe('сообщения наставника', () => {
  it('открывает тред, отмечает прочитанным и отвечает с добавлением в справочник', async () => {
    loginAs('mentor_igor')
    const router = renderApp('/inbox')
    expect(await screen.findByRole('button', { name: 'Анна, новых: 2' }, T)).toBeInTheDocument()
    expect(await screen.findByRole('log', { name: 'Переписка с Анна' }, T)).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: 'Анна, новых: 2' })).not.toBeInTheDocument(),
    )
    expect(screen.getByText('Стажер еще не открыл')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Обсудим на 1:1' }))
    await userEvent.click(screen.getByLabelText(/Добавить ответ в справочник/))
    await userEvent.click(screen.getByRole('button', { name: 'Отправить' }))
    expect(await toastText(/AI-buddy добавил его в справочник/)).toBeInTheDocument()
    expect(within(screen.getByRole('log')).getByText('Обсудим на 1:1')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Лиза, новых: 1' }))
    expect(router.state.location.search).toEqual({ intern: 'u-liza' })
    expect(
      await within(await screen.findByRole('log', { name: 'Переписка с Лиза' }, T)).findByText(
        'Привет! Во сколько завтра встреча 1:1?',
      ),
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('link', { name: 'Материалы и справочник' }))
    expect(await screen.findByText('Ответ наставника: Обсудим на 1:1', {}, T)).toBeInTheDocument()
  })

  it('обычный ответ без справочника', async () => {
    loginAs('mentor_igor')
    renderApp('/inbox?intern=u-dima')
    const box = await screen.findByLabelText('Ответ стажеру', {}, T)
    await userEvent.type(box, 'Отлично, вливай{Enter}')
    expect(await toastText(/Дмитрий получит ответ и уведомление/)).toBeInTheDocument()
  })

  it('пустой список и ошибка ответа', async () => {
    server.use(http.get(`${API_URL}/api/mentor/threads`, () => HttpResponse.json([])))
    loginAs('mentor_igor')
    renderApp('/inbox')
    expect(await screen.findByText(/AI-buddy справляется сам/, {}, T)).toBeInTheDocument()
  })

  it('ошибка отправки ответа', async () => {
    server.use(
      http.post(`${API_URL}/api/mentor/threads/:id/messages`, () =>
        HttpResponse.json({ error: { code: 'internal_error', message: 'x' } }, { status: 500 }),
      ),
    )
    loginAs('mentor_igor')
    renderApp('/inbox?intern=u-dima')
    await userEvent.type(await screen.findByLabelText('Ответ стажеру', {}, T), 'ок{Enter}')
    expect(await screen.findByRole('alert', {}, T)).toBeInTheDocument()
  })
})

describe('план адаптации', () => {
  it('добавляет, меняет и удаляет задачу', async () => {
    loginAs('mentor_igor')
    renderApp('/plan')
    const stage = await screen.findByRole('region', { name: 'Этап Доступы' }, T)
    expect(within(stage).getByText('4 задачи')).toBeInTheDocument()
    expect(within(stage).getAllByText('AI-buddy оформит заявку на доступ')).toHaveLength(2)

    await userEvent.click(within(stage).getByRole('button', { name: 'Задача' }))
    const dialog = await screen.findByRole('dialog', {}, T)
    await userEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(await within(dialog).findByText('Введи название', {}, T)).toBeInTheDocument()
    await userEvent.type(within(dialog).getByLabelText('Название задачи'), 'Доступ к дашбордам')
    await userEvent.type(within(dialog).getByLabelText('День стажировки'), '9')
    await userEvent.selectOptions(
      within(dialog).getByLabelText('Что может сделать AI-buddy'),
      'access_request',
    )
    await userEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(await toastText(/Изменение уже в «Моем пути»/)).toBeInTheDocument()
    expect(await within(stage).findByText('Доступ к дашбордам', {}, T)).toBeInTheDocument()
    expect(within(stage).getByText('5 задач')).toBeInTheDocument()

    await userEvent.click(
      within(stage).getByRole('button', { name: 'Изменить задачу «Доступ к дашбордам»' }),
    )
    const edit = await screen.findByRole('dialog', {}, T)
    const title = within(edit).getByLabelText('Название задачи')
    expect(title).toHaveValue('Доступ к дашбордам')
    await userEvent.clear(title)
    await userEvent.type(title, 'Доступ к дашбордам метрик')
    await userEvent.click(within(edit).getByRole('button', { name: 'Сохранить' }))
    expect(await within(stage).findByText('Доступ к дашбордам метрик', {}, T)).toBeInTheDocument()

    await userEvent.click(
      within(stage).getByRole('button', { name: 'Удалить задачу «Доступ к дашбордам метрик»' }),
    )
    expect(await toastText(/удалена/)).toBeInTheDocument()
    await waitFor(() =>
      expect(within(stage).queryByText('Доступ к дашбордам метрик')).not.toBeInTheDocument(),
    )
  })

  it('добавляет этап, двигает и удаляет его', async () => {
    loginAs('lead_oleg')
    renderApp('/plan')
    await screen.findByRole('region', { name: 'Этап Доступы' }, T)
    await userEvent.click(screen.getByRole('button', { name: 'Этап' }))
    const dialog = await screen.findByRole('dialog', {}, T)
    await userEvent.type(within(dialog).getByLabelText('Название этапа'), 'Итоги стажировки')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Цвет розовый' }))
    expect(within(dialog).getByRole('button', { name: 'Цвет розовый' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await userEvent.click(within(dialog).getByRole('button', { name: 'Добавить этап' }))
    expect(
      await screen.findByRole('region', { name: 'Этап Итоги стажировки' }, T),
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Сдвинуть «Итоги стажировки» левее' }))
    expect(await toastText(/перемещен/)).toBeInTheDocument()
    await userEvent.click(
      screen.getByRole('button', { name: 'Сдвинуть «Знакомство с командой» правее' }),
    )

    await userEvent.click(screen.getByRole('button', { name: 'Удалить этап «Итоги стажировки»' }))
    const confirm = await screen.findByRole('dialog', { name: 'Удалить этап?' }, T)
    expect(within(confirm).getByText(/0 задач пропадут/)).toBeInTheDocument()
    await userEvent.click(within(confirm).getByRole('button', { name: 'Удалить' }))
    await waitFor(() =>
      expect(
        screen.queryByRole('region', { name: 'Этап Итоги стажировки' }),
      ).not.toBeInTheDocument(),
    )
  })

  it('ошибка сохранения задачи остается в окне', async () => {
    server.use(
      http.post(`${API_URL}/api/console/stages/:id/tasks`, () =>
        HttpResponse.json({ error: { code: 'validation_error', message: 'bad' } }, { status: 400 }),
      ),
    )
    loginAs('mentor_igor')
    renderApp('/plan')
    const stage = await screen.findByRole('region', { name: 'Этап Доступы' }, T)
    await userEvent.click(within(stage).getByRole('button', { name: 'Задача' }))
    const dialog = await screen.findByRole('dialog', {}, T)
    await userEvent.type(within(dialog).getByLabelText('Название задачи'), 'X')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(await within(dialog).findByRole('alert', {}, T)).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Отмена' }))
  })

  it('пустой шаблон и переключение автоматизаций', async () => {
    server.use(http.get(`${API_URL}/api/console/templates`, () => HttpResponse.json([])))
    loginAs('mentor_igor')
    renderApp('/plan')
    expect(await screen.findByText(/В шаблоне пока нет этапов/, {}, T)).toBeInTheDocument()
    expect(screen.queryByRole('switch', { name: 'Напоминать согласующему о заявке' })).toBeNull()
    cleanup()
    renderApp('/profile')
    const sw = await screen.findByRole('switch', { name: 'Напоминать согласующему о заявке' }, T)
    expect(sw).not.toBeChecked()
    await userEvent.click(sw)
    expect(await toastText(/Включено: напоминать согласующему/)).toBeInTheDocument()
    await waitFor(() => expect(sw).toBeChecked())
  })
})

describe('материалы', () => {
  it('загружает ссылку, проверяет поля и удаляет материал', async () => {
    loginAs('mentor_igor')
    renderApp('/materials')
    expect(await screen.findByText('Правила код-ревью команды', {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Загрузить материал' }))
    const dialog = await screen.findByRole('dialog', {}, T)
    await userEvent.click(within(dialog).getByRole('button', { name: 'Загрузить' }))
    expect(
      await within(dialog).findByText('Выбери файл или вставь ссылку', {}, T),
    ).toBeInTheDocument()

    await userEvent.click(within(dialog).getByText('Вставить ссылку'))
    await userEvent.type(within(dialog).getByLabelText('Ссылка'), 'https://wiki.example/review')
    await userEvent.type(within(dialog).getByLabelText('Название'), 'Как проходит ревью')
    await userEvent.selectOptions(within(dialog).getByLabelText('Показывать на этапе'), '03')
    await userEvent.click(within(dialog).getByLabelText('AI-buddy использует материал в ответах'))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Загрузить' }))
    expect(await toastText(/Стажеры получили уведомление/)).toBeInTheDocument()
    expect(await screen.findByText('Как проходит ревью', {}, T)).toBeInTheDocument()
    expect(screen.getAllByText('только чтение')).toHaveLength(2)

    await userEvent.click(screen.getByRole('button', { name: 'Удалить «Как проходит ревью»' }))
    await waitFor(() => expect(screen.queryByText('Как проходит ревью')).not.toBeInTheDocument())
  })

  it('файл подставляет название; ошибка загрузки показана в окне', async () => {
    loginAs('mentor_igor')
    renderApp('/materials')
    await userEvent.click(
      await screen.findByRole('button', { name: 'Загрузить материал: выбрать файл' }, T),
    )
    const dialog = await screen.findByRole('dialog', {}, T)
    const file = new File(['# ревью'], 'pravila_review.pdf', { type: 'application/pdf' })
    await userEvent.upload(dialog.querySelector<HTMLInputElement>('input[type=file]')!, file)
    expect(within(dialog).getByLabelText('Название')).toHaveValue('pravila review')
    await userEvent.click(within(dialog).getByLabelText('Сообщить стажерам о новом материале'))
    // в jsdom multipart с файлом не отправляется – проверяем, что ошибка видна и окно не закрылось
    await userEvent.click(within(dialog).getByRole('button', { name: 'Загрузить' }))
    expect(await within(dialog).findByRole('alert', {}, T)).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Отмена' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('ошибка загрузки списка', async () => {
    server.use(
      http.get(`${API_URL}/api/materials`, () =>
        HttpResponse.json({ error: { code: 'internal_error', message: 'x' } }, { status: 500 }),
      ),
    )
    loginAs('mentor_igor')
    renderApp('/materials')
    expect(await screen.findByRole('button', { name: 'Повторить' }, T)).toBeInTheDocument()
  })
})

describe('расписание', () => {
  it('переносит и отменяет встречу, стажер видит изменения в чате', async () => {
    loginAs('mentor_igor')
    renderApp('/schedule')
    await userEvent.click(await screen.findByRole('button', { name: /^1:1 с Анной, чт 15:00/ }, T))
    expect(screen.getByText('С кем: Анна')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Перенести на час' }))
    expect(await toastText(/Анна: встреча перенесена на 16:00/)).toBeInTheDocument()
    expect(
      await screen.findByRole('button', { name: /^1:1 с Анной, чт 16:00/ }, T),
    ).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Отменить' }))
    expect(await toastText(/отменена, Анна получит уведомление/)).toBeInTheDocument()
    expect(screen.getByText(/Выбери встречу в календаре/)).toBeInTheDocument()
  })

  it('назначает встречу и листает недели', async () => {
    loginAs('mentor_igor')
    renderApp('/schedule')
    await screen.findByRole('button', { name: /^Демо команды/ }, T)
    await userEvent.click(screen.getByRole('button', { name: 'Встреча' }))
    const dialog = await screen.findByRole('dialog', {}, T)
    await waitFor(() =>
      expect(within(dialog).getByLabelText('Стажер')).toHaveValue(
        (within(dialog).getByLabelText('Стажер') as HTMLSelectElement).options[0]!.value,
      ),
    )
    await userEvent.selectOptions(within(dialog).getByLabelText('Стажер'), 'u-liza')
    await userEvent.selectOptions(within(dialog).getByLabelText('День'), '2')
    await userEvent.selectOptions(within(dialog).getByLabelText('Время'), '10')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Назначить' }))
    expect(await toastText(/Лиза получит приглашение/)).toBeInTheDocument()
    expect(
      await screen.findByRole('button', { name: /^1:1 · Лиза, ср 10:00/ }, T),
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Встреча' }))
    const d2 = await screen.findByRole('dialog', {}, T)
    await userEvent.selectOptions(within(d2).getByLabelText('Стажер'), 'all')
    await userEvent.click(within(d2).getByLabelText('Отправить приглашение и напомнить за час'))
    await userEvent.click(within(d2).getByRole('button', { name: 'Назначить' }))
    expect(await toastText(/без уведомления/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Следующая неделя' }))
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: /^Демо команды/ })).not.toBeInTheDocument(),
    )
    await userEvent.click(screen.getByRole('button', { name: 'Прошлая неделя' }))
    expect(await screen.findByRole('button', { name: /^Демо команды/ }, T)).toBeInTheDocument()
  })

  it('ошибка назначения встречи видна в окне', async () => {
    server.use(
      http.post(`${API_URL}/api/mentor/meetings`, () =>
        HttpResponse.json({ error: { code: 'internal_error', message: 'x' } }, { status: 500 }),
      ),
    )
    loginAs('mentor_igor')
    renderApp('/schedule')
    await userEvent.click(await screen.findByRole('button', { name: 'Встреча' }, T))
    const dialog = await screen.findByRole('dialog', {}, T)
    await userEvent.type(within(dialog).getByLabelText('Тема'), 'Созвон')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Назначить' }))
    expect(await within(dialog).findByRole('alert', {}, T)).toBeInTheDocument()
  })
})

describe('mentorUi', () => {
  it('склоняет «задача»', () => {
    expect([1, 2, 5, 11, 12, 21, 22, 25].map(tasksWord)).toEqual([
      '1 задача',
      '2 задачи',
      '5 задач',
      '11 задач',
      '12 задач',
      '21 задача',
      '22 задачи',
      '25 задач',
    ])
  })

  it('понедельник недели и подпись времени', () => {
    const sunday = new Date(2026, 10, 15, 13, 0)
    expect(mondayOf(sunday).getDate()).toBe(9)
    expect(addDays(mondayOf(sunday), 3).getDate()).toBe(12)
    const now = new Date(2026, 10, 10, 18, 0)
    expect(timeLabel(new Date(2026, 10, 10, 14, 5).toISOString(), now)).toBe('14:05')
    expect(timeLabel(new Date(2026, 10, 9, 14, 5).toISOString(), now)).toMatch(/9 нояб.*14:05/)
  })
})
