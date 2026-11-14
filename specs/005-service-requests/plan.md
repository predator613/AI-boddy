# 005. План: заявки в отделы

## Последовательность

```mermaid
sequenceDiagram
    actor S as Стажер
    participant A as Agent (backend)
    participant L as LLM
    participant DB as PostgreSQL
    participant M as mock-services (Jira Service Desk)

    S->>A: «нужен монитор» / «нужно оборудование»
    alt предмет не назван, есть отдел
        A->>DB: каталог отдела
        A-->>S: «Что именно нужно? В «Отдел оборудования» можно заказать: …»
    else
        A->>L: сообщение + инструменты
        L-->>A: prepare_service_request(item, comment)
        A->>DB: синоним → позиция каталога, открытая заявка?, черновик
        A-->>S: карточка (что нужно, комментарий) + отдел и срок
    end
    S->>A: «Подтвердить»
    A->>M: POST /v1/tickets (kind other)
    M-->>A: SD-8001 waiting
    A->>DB: заявка waiting, уведомление наставнику
    A-->>S: «Заявка SD-8001 создана»
    loop синхронизация
        A->>M: GET /v1/tickets/SD-8001
        A->>DB: done, сообщение в чат
    end
```

## Решения

- **Каталог и синонимы в базе** (`catalog_items`, `catalog_aliases`): при нескольких совпадениях выигрывает самый длинный синоним («зарядка для ноутбука» → другое оборудование, а не ноутбук).
- **Уточнение до модели.** Общая просьба к отделу («оборудование», «бухгалтерия») без предмета разбирается правилом: модель на такие запросы часто отвечала текстом без заявки.
- **Позиция «Другое оборудование»** вместо отказа: заявка все равно доходит до отдела, подробности в комментарии.
- **Один адаптер к системе заявок** для доступов, отделов и обращений в поддержку; при внедрении заменяется на API Jira Service Desk или MCP-сервер.

## Данные

Миграции 013 (отделы, каталог, синонимы, `service_request_details`), 021 (другое оборудование и новые синонимы).

## Код

- `internal/services/agent.go` – `prepareService`, `departmentHelp`, `reServiceAsk`; `agent_service.go` – `catalogHelp`, уведомление наставнику.
- `internal/repository/requests.go` – `ResolveItem`, `CatalogItem`, `CreateServiceDraft`.
- Frontend: `pages/SupportPage.tsx` (кнопка «Оформить заявку»), `pages/RequestsPage.tsx`.
