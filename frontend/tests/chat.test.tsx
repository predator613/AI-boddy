import { cleanup, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { API_URL } from '../src/api/client'
import { server } from '../src/mocks/server'
import { loginAs, renderApp } from './test-utils'

const T = { timeout: 3000 }

async function send(text: string) {
  const box = await screen.findByLabelText('Сообщение')
  await userEvent.type(box, text)
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }))
}

describe('Чат', () => {
  beforeEach(() => loginAs('intern_maria'))

  it('пустой чат приветствует и предлагает вопросы', async () => {
    renderApp('/chat')
    expect(await screen.findByText('Привет, Мария! С чего начнем?', {}, T)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Новый диалог' })).toBeDisabled()
  })

  it('пока помощник думает – статус ожидания, потом ответ с источником', async () => {
    renderApp('/chat')
    await send('Как устроено код-ревью?')
    expect(await screen.findByRole('status')).toHaveTextContent('AI-buddy думает')
    expect(await screen.findByText(/ревьюер оставляет комментарии/i, {}, T)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Источник: Код, ревью и релизы/ })).toBeInTheDocument()
  })

  it('Enter отправляет, Shift+Enter переносит строку', async () => {
    renderApp('/chat')
    const box = await screen.findByLabelText('Сообщение')
    await userEvent.type(box, 'Кто мой{Shift>}{Enter}{/Shift}наставник?')
    expect(box).toHaveValue('Кто мой\nнаставник?')
    await userEvent.type(box, '{Enter}')
    expect(await screen.findByText(/Твой наставник – Игорь/, {}, T)).toBeInTheDocument()
  })

  it('подсказка-чип отправляет вопрос, отгул можно отменить', async () => {
    renderApp('/chat')
    await userEvent.click(await screen.findByRole('button', { name: 'Как оформить отгул?' }, T))
    expect(await screen.findByText('Создать заявку на отгул?', {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Отмена' }))
    expect(await screen.findByText(/не создаю/, {}, T)).toBeInTheDocument()
    expect(screen.getByText('Отменено')).toBeInTheDocument()
  })

  it('недоступное стажеру помощник не выполняет', async () => {
    renderApp('/chat')
    await send('Дай доступ к логам прода')
    expect(await screen.findByText(/стажерам не выдается/, {}, T)).toBeInTheDocument()
  })

  it('прочие вопросы помощник предлагает передать наставнику', async () => {
    renderApp('/chat')
    await send('Где взять печеньки?')
    expect(await screen.findByText(/Передать вопрос Игорю/, {}, T)).toBeInTheDocument()
    await send('Сколько ждать согласования?')
    expect(await screen.findByText(/сам напомню Артемию/, {}, T)).toBeInTheDocument()
    await send('Какой следующий шаг?')
    expect(await screen.findByText(/Подготовить заявку\?/, {}, T)).toBeInTheDocument()
  })

  it('ошибка сервера – сообщение и «Повторить»', async () => {
    server.use(
      http.post(`${API_URL}/api/chat/messages`, () => new HttpResponse(null, { status: 500 }), {
        once: true,
      }),
    )
    renderApp('/chat')
    await send('Нужен gitlab')
    expect(await screen.findByRole('alert', {}, T)).toHaveTextContent('Не удалось загрузить')
    await userEvent.click(screen.getByRole('button', { name: 'Повторить' }))
    expect(await screen.findByText('Создать заявку на доступ?', {}, T)).toBeInTheDocument()
  })

  it('ошибка подтверждения показывается', async () => {
    server.use(
      http.post(`${API_URL}/api/chat/actions/:id/confirm`, () =>
        HttpResponse.json({ error: { code: 'not_found', message: 'x' } }, { status: 404 }),
      ),
    )
    renderApp('/chat')
    await send('gitlab')
    await userEvent.click(await screen.findByRole('button', { name: 'Подтвердить' }, T))
    expect(await screen.findByRole('alert', {}, T)).toHaveTextContent('Ничего не нашлось')
  })

  it('«Новый диалог» очищает переписку', async () => {
    renderApp('/chat?q=Кто мой наставник?')
    expect(await screen.findByText(/Твой наставник – Игорь/, {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Новый диалог' }))
    expect(await screen.findByText('Привет, Мария! С чего начнем?', {}, T)).toBeInTheDocument()
    expect(screen.queryByText(/Твой наставник – Игорь/)).not.toBeInTheDocument()
  })

  it('переписка восстанавливается после перезагрузки страницы', async () => {
    renderApp('/chat')
    await send('Как оформить отгул?')
    const card = await screen.findByRole('button', { name: 'Отмена' }, T)
    await userEvent.click(card)
    expect(await screen.findByText(/заявку не создаю|не создаю/i, {}, T)).toBeInTheDocument()

    // «Перезагрузка»: новое приложение с пустым кешем берет переписку из backend.
    cleanup()
    renderApp('/chat')
    expect(await screen.findByText(/не создаю/i, {}, T)).toBeInTheDocument()
    // вопрос из истории плюс такая же подсказка-чип
    expect(screen.getAllByText('Как оформить отгул?')).toHaveLength(2)
    expect(screen.queryByRole('button', { name: 'Подтвердить' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Новый диалог' })).toBeEnabled()
  })
})

describe('история диалогов', () => {
  it('прошлый диалог открывается только для чтения, «Новый диалог» переносит текущий в историю', async () => {
    loginAs('intern_maria')
    renderApp('/chat')
    const side = await screen.findByRole('complementary', { name: 'Диалоги' }, { timeout: 3000 })
    await userEvent.click(await within(side).findByRole('button', { name: /Отгул на 14 ноября/ }))
    expect(
      await screen.findByText('Заявка на отгул создана, Артемий ее одобрил.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Это прошлый диалог, он только для чтения.')).toBeInTheDocument()
    expect(screen.queryByLabelText('Сообщение')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Вернуться к текущему' }))

    await userEvent.type(await screen.findByLabelText('Сообщение'), 'Что такое код-ревью?{Enter}')
    await screen.findByText(/ревьюер оставляет комментарии/i, {}, { timeout: 3000 })
    await userEvent.click(within(side).getByRole('button', { name: 'Новый диалог' }))
    expect(
      await within(side).findByRole('button', { name: /Что такое код-ревью\?/ }, { timeout: 3000 }),
    ).toBeInTheDocument()
  })

  it('без ручки истории на backend остается только текущий диалог', async () => {
    server.use(
      http.get(`${API_URL}/api/chat/dialogs`, () => new HttpResponse(null, { status: 404 })),
    )
    loginAs('intern_maria')
    renderApp('/chat')
    const side = await screen.findByRole('complementary', { name: 'Диалоги' }, { timeout: 3000 })
    expect(within(side).getByRole('button', { name: /Текущий диалог/ })).toBeInTheDocument()
    expect(within(side).queryByRole('button', { name: /Отгул/ })).not.toBeInTheDocument()
  })
})

describe('карточка «Заявка создана» с ключом Jira', () => {
  it('показывает ключ и ссылку из backend', async () => {
    const { ChatMessageView } = await import('../src/components/chat/ChatMessageView')
    const { renderWithProviders } = await import('./test-utils')
    const { RouterProvider, createMemoryHistory, createRootRoute, createRouter } =
      await import('@tanstack/react-router')
    const msg = {
      id: 'm1',
      role: 'assistant' as const,
      kind: 'result' as const,
      text: '',
      source: null,
      confirm: null,
      mood: 'happy' as const,
      result: {
        requestId: 'r1',
        number: '1042',
        title: 'Доступ к GitLab',
        status: 'waiting' as const,
        external: {
          system: 'Jira Service Desk',
          key: 'ACCESS-8001',
          url: 'http://localhost:8090/browse/ACCESS-8001',
        },
      },
    }
    const router = createRouter({
      routeTree: createRootRoute({ component: () => <ChatMessageView message={msg} /> }),
      history: createMemoryHistory(),
    })
    renderWithProviders(<RouterProvider router={router} />)
    expect(await screen.findByText('Заявка ACCESS-8001 создана')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'ACCESS-8001 в Jira Service Desk' })).toHaveAttribute(
      'href',
      'http://localhost:8090/browse/ACCESS-8001',
    )
  })
})

describe('стиль ответа из профиля', () => {
  it('уходит в запросе к помощнику', async () => {
    const { useSettingsStore } = await import('../src/store/settingsStore')
    let style: unknown
    server.use(
      http.post(`${API_URL}/api/chat/messages`, async ({ request }) => {
        style = ((await request.json()) as { style?: string }).style
        return HttpResponse.json({ messages: [] })
      }),
    )
    loginAs('intern_maria')
    useSettingsStore.getState().setAnswerStyle('short')
    const { sendMessage } = await import('../src/api/assistant')
    await sendMessage({ text: 'что такое спринт' })
    expect(style).toBe('short')
  })
})
