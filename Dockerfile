# Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm run build

# Build backend
FROM golang:1.24-alpine AS backend
WORKDIR /app
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/web/dist ./internal/web/dist

RUN CGO_ENABLED=1 go build -o ampmanager ./cmd/server

# Runtime
FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=backend /app/ampmanager .

EXPOSE 16823

VOLUME ["/app/data"]

CMD ["./ampmanager"]
