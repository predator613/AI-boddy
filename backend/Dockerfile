# syntax=docker/dockerfile:1

# ---- 1. Зависимости ----
FROM golang:1.25-alpine AS deps                 
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

# ---- 2. Исходники ----
FROM deps AS source
COPY . .
RUN go mod tidy

# ---- 3. Тесты с отчетом о покрытии ----
# Имитация корпоративных систем (mockservices, cmd/*) в расчет покрытия не входит.
FROM source AS test
CMD ["sh", "-c", "mkdir -p /coverage && PKGS=$(go list ./... | grep -v -e /mockservices -e /cmd/ | paste -sd, -) && go test -p 1 -count=1 -covermode=atomic -coverpkg=$PKGS -coverprofile=/coverage/coverage.out ./... && go tool cover -func=/coverage/coverage.out | tail -n 1 && go tool cover -html=/coverage/coverage.out -o /coverage/coverage.html"]

# ---- 4. Сборка бинарника ----
FROM source AS build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mockservices ./cmd/mockservices

# ---- 5. Итоговый образ ----
FROM alpine:3.20 AS runtime
RUN adduser -D -u 10001 app
COPY --from=build /out/server /app/server
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]

# ---- 6. Имитация корпоративной системы заявок (отдельный контейнер mock-services) ----
FROM alpine:3.20 AS mockservices
RUN adduser -D -u 10001 app
COPY --from=build /out/mockservices /app/mockservices
USER app
EXPOSE 8090
ENTRYPOINT ["/app/mockservices"]
