# syntax=docker/dockerfile:1

# ---- 1. Зависимости (кешируются, пока не меняется go.mod/go.sum) ----
FROM golang:1.25-alpine AS deps
# ---- 1. Зависимости ----
FROM golang:1.25-alpine AS deps                 
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

# ---- 2. Исходники ----
FROM deps AS source
COPY . .
# Досчитывает go.sum, если его забыли закоммитить. Версии берутся из go.mod и не меняются.
RUN go mod tidy

# ---- 3. Тесты с отчетом о покрытии: docker compose --profile test run --rm backend-tests ----
# ---- 3. Тесты с отчетом о покрытии ----
FROM source AS test
CMD ["sh", "-c", "mkdir -p /coverage && go test -p 1 -count=1 -covermode=atomic -coverpkg=./... -coverprofile=/coverage/coverage.out ./... && go tool cover -func=/coverage/coverage.out | tail -n 1 && go tool cover -html=/coverage/coverage.out -o /coverage/coverage.html"]

# ---- 4. Сборка бинарника ----
FROM source AS build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- 5. Маленький итоговый образ ----
# ---- 5. Итоговый образ ----
FROM alpine:3.20 AS runtime
RUN adduser -D -u 10001 app
COPY --from=build /out/server /app/server
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
ENTRYPOINT ["/app/server"]