# syntax=docker/dockerfile:1
FROM node:20-alpine AS frontend-build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
COPY backend/api ../backend/api
RUN npm run generate:api
RUN npm run build

FROM golang:1.24-alpine AS backend-build
WORKDIR /app
RUN apk add --no-cache make
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-build /app/dist ./internal/web/dist
RUN go build -trimpath -ldflags="-s -w" -o /out/prometheus ./cmd/prometheus

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata nmap openssh-client iproute2
RUN adduser -D -u 10001 prometheus
USER prometheus
WORKDIR /app
COPY --from=backend-build /out/prometheus /app/prometheus
COPY --from=backend-build /app/db/migrations /app/db/migrations
EXPOSE 8180
ENTRYPOINT ["/app/prometheus"]
