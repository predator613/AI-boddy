# Установка и запуск backend AI-buddy на Windows 10/11 (VS Code + Docker)

## Шаг 0. Проверить виртуализацию
1. Диспетчер задач → Производительность → ЦП → строка «Виртуализация: Включено».
2. Если «Отключено» – включить в BIOS/UEFI (Intel VT-x или AMD SVM), перезагрузиться.

## Шаг 1. WSL 2 (на нем работает Docker Desktop)
1. PowerShell от имени администратора:
   ```
   wsl --install
   ```
2. Перезагрузить компьютер. Если откроется окно Ubuntu – придумать логин и пароль (любые).
3. Проверить:
   ```
   wsl --update
   wsl --status
   ```
   Должно быть «Версия по умолчанию: 2».

## Шаг 2. Docker Desktop
1. Скачать Docker Desktop for Windows с docker.com, установить. Оставить галочку «Use WSL 2 instead of Hyper-V».
2. Перезагрузиться, запустить Docker Desktop, принять соглашение. Входить в аккаунт не обязательно (Skip).
3. Дождаться внизу слева зеленой надписи «Engine running».
4. В обычном терминале:
   ```
   docker version
   docker compose version
   docker run --rm hello-world
   ```
   Последняя команда должна напечатать «Hello from Docker!».
5. Если образы не скачиваются (timeout, 403 или «failed to resolve reference»): Docker Desktop → Settings → Docker Engine, добавить в JSON строку
   `"registry-mirrors": ["https://mirror.gcr.io"]`, нажать Apply & restart.

## Шаг 3. Git
1. Установить Git for Windows (git-scm.com).
2. Один раз выполнить:
   ```
   git config --global core.autocrlf input
   git config --global user.name "Мария Воронина"
   git config --global user.email "почта"
   ```
   Зачем: Windows по умолчанию ставит переводы строк CRLF, а в Linux-контейнерах они ломают скрипты. В проекте еще есть `.gitattributes`, который принудительно ставит LF.

## Шаг 4. Go
1. Скачать установщик .msi с go.dev/dl (последняя стабильная версия), установить с настройками по умолчанию.
2. Открыть новый терминал: `go version`.
3. Нужен для подсказок в VS Code, отладки и команды `go mod tidy`. Сам запуск через Docker работает и без него.

## Шаг 5. VS Code
1. Открыть папку `ai-buddy` (File → Open Folder).
2. Справа внизу появится «Установить рекомендуемые расширения» – согласиться. Будут поставлены: Go, Docker, WSL, REST Client, EditorConfig.
3. Ctrl+Shift+P → «Go: Install/Update Tools» → отметить все → OK.
4. Терминал: Ctrl+` (PowerShell). Важно: в Windows PowerShell команда `curl` – это не настоящий curl. Пишите `curl.exe` или используйте `backend/api.http`.

## Шаг 6. Первый запуск проекта
```
cd backend
go mod tidy          # создаст go.sum, его нужно закоммитить
cd ..
docker compose up --build -d
docker compose ps
```
Первая сборка идет 2–5 минут (скачиваются образы и Go-модули). Дальше быстрее.

Проверка:
- браузер: http://localhost:8080/api/health → `{"status":"ok"}`
- http://localhost:8080/api/auth/test-users → 4 пользователя
- в VS Code открыть `backend/api.http`, нажать «Send Request» над запросом входа, потом над `/me`

Логи: `docker compose logs -f backend`. Остановить: `docker compose down`. Сбросить базу: `docker compose down -v`.

## Шаг 7. Тесты и покрытие
```
docker compose --profile test run --rm backend-tests
```
В конце будет строка `total: (statements) 89.x%`. Отчет – `backend/coverage/coverage.html`, открыть в браузере.

Из VS Code: при запущенном `docker compose up` можно нажимать «run test» над функциями в `*_test.go` – адрес тестовой базы уже прописан в `.vscode/settings.json`.

## Шаг 8. Отладка backend в VS Code
1. `docker compose stop backend` (чтобы освободить порт 8080; postgres пусть работает).
2. F5 → «Backend (локально, база в Docker)». Можно ставить точки останова.
3. Вернуть контейнер: `docker compose start backend`.

## Частые проблемы
- `port is already allocated` на 5432 – на компьютере уже стоит свой PostgreSQL. В `.env` поставить `POSTGRES_PORT=5433` (и поменять порт в `.vscode/settings.json` и `launch.json`).
- `database "ai_buddy_test" does not exist` – `docker compose down -v`, затем снова `up`.
- `go mod tidy` долго висит – повторить; при проблемах с сетью: `go env -w GOPROXY=https://goproxy.io,direct`.
- Docker Desktop не стартует – проверить шаги 0–1, `wsl --update`, перезагрузка.
