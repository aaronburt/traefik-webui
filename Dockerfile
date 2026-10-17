FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY --from=web /src/internal/server/dist ./internal/server/dist
RUN CGO_ENABLED=0 go build -o /traefik-webui ./cmd/traefik-webui

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /traefik-webui /traefik-webui
ENV DATA_DIR=/data
VOLUME /data
EXPOSE 8081
ENTRYPOINT ["/traefik-webui"]
