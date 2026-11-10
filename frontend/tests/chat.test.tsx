import { screen } from '@testing-library/react'
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
  beforeEach(() => loginAs('intern_anna'))

  it('пустой чат приветствует и предлагает вопросы', async () => {
    renderApp('/chat')
    expect(await screen.findByText('Привет, Анна! С чего начнем?', {}, T)).toBeInTheDocument()
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
    expect(await screen.findByText(/сам напомню Олегу/, {}, T)).toBeInTheDocument()
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
    expect(screen.queryByText(/Твой наставник – Игорь/)).not.toBeInTheDocument()
  })
})
