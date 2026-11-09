# 1. Сборка
FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_URL=http://localhost:8080
ARG VITE_USE_MOCKS=false
ENV VITE_API_URL=$VITE_API_URL VITE_USE_MOCKS=$VITE_USE_MOCKS
RUN npm run build

# 2. Тесты + отчет о покрытии (docker compose --profile test run --rm frontend-tests)
FROM build AS test
CMD ["sh", "-c", "npm run coverage && rm -rf /coverage/* && cp -r coverage/. /coverage/"]

# 3. Раздача статики
FROM nginx:1.27-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
