// API страницы «Мой путь» и консоли наставника. Все запросы идут через общий client.ts.
//
// Ожидается, что client.ts экспортирует apiFetch(path, init) -> Promise<unknown>:
// подставляет VITE_API_URL и токен, бросает ошибку на не-2xx, на 401 делает logout,
// возвращает разобранный JSON (или undefined для ответа 204).
// Если функция в client.ts называется иначе – поменяйте только строку импорта ниже.
import { apiFetch } from './client';
import { z } from 'zod';
import {
  InternSummary,
  Meeting,
  PathView,
  ResourceTemplate,
  StageTemplate,
  TaskTemplate,
  type TMeetingInput,
  type TResourceInput,
  type TSetStatusInput,
  type TStageInput,
  type TTaskInput,
} from '../schemas/path';

const json = (method: string, body?: unknown): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: body === undefined ? undefined : JSON.stringify(body),
});

async function get<T>(schema: z.ZodType<T>, path: string, init?: RequestInit): Promise<T> {
  return schema.parse(await apiFetch(path, init));
}

// ---------- стажер ----------

/** GET /api/path – весь экран «Мой путь» текущего стажера. */
export const getMyPath = () => get(PathView, '/api/path');

/** POST /api/path/tasks/{id}/complete – кнопка «Уже сделано». Возвращает обновленный путь. */
export const completeTask = (taskId: string) =>
  get(PathView, `/api/path/tasks/${encodeURIComponent(taskId)}/complete`, json('POST'));

// ---------- наставник и руководитель ----------

/** GET /api/interns – мои стажеры с прогрессом. */
export const getInterns = () => get(z.array(InternSummary), '/api/interns');

/** GET /api/interns/{id}/path – путь конкретного стажера (тот же формат, что у стажера). */
export const getInternPath = (internId: string) =>
  get(PathView, `/api/interns/${encodeURIComponent(internId)}/path`);

/** PUT /api/interns/{id}/tasks/{taskId} – поменять статус задачи стажера. */
export const setInternTaskStatus = (internId: string, taskId: string, input: TSetStatusInput) =>
  get(
    PathView,
    `/api/interns/${encodeURIComponent(internId)}/tasks/${encodeURIComponent(taskId)}`,
    json('PUT', input),
  );

/** POST /api/interns/{id}/meetings – назначить встречу. */
export const createMeeting = (internId: string, input: TMeetingInput) =>
  get(Meeting, `/api/interns/${encodeURIComponent(internId)}/meetings`, json('POST', input));

// ---------- консоль наставника: шаблоны ----------

export const getTemplates = () => get(z.array(StageTemplate), '/api/console/templates');

export const getActionKinds = () =>
  get(z.array(z.object({ kind: z.string(), description: z.string() })), '/api/console/action-kinds');

export const createStage = (input: TStageInput) =>
  get(StageTemplate, '/api/console/stages', json('POST', input));

export const updateStage = (stageId: string, input: TStageInput & { position: number }) =>
  get(StageTemplate, `/api/console/stages/${encodeURIComponent(stageId)}`, json('PUT', input));

export const deleteStage = (stageId: string) =>
  apiFetch(`/api/console/stages/${encodeURIComponent(stageId)}`, json('DELETE'));

export const createTask = (stageId: string, input: TTaskInput) =>
  get(TaskTemplate, `/api/console/stages/${encodeURIComponent(stageId)}/tasks`, json('POST', input));

export const updateTask = (taskId: string, input: TTaskInput & { position: number }) =>
  get(TaskTemplate, `/api/console/tasks/${encodeURIComponent(taskId)}`, json('PUT', input));

export const deleteTask = (taskId: string) =>
  apiFetch(`/api/console/tasks/${encodeURIComponent(taskId)}`, json('DELETE'));

export const createResource = (stageId: string, input: TResourceInput) =>
  get(ResourceTemplate, `/api/console/stages/${encodeURIComponent(stageId)}/resources`, json('POST', input));

export const deleteResource = (resourceId: string) =>
  apiFetch(`/api/console/resources/${encodeURIComponent(resourceId)}`, json('DELETE'));
