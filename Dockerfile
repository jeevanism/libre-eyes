# Build the React static bundle separately from the Go runtime.
FROM node:24-alpine AS web-build
WORKDIR /src
COPY . .
RUN npm --prefix web ci \
    && npm --prefix web run build

FROM golang:1.25-alpine AS api-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/visionopus-api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/visionopus-migrate ./cmd/migrate \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/visionopus-devseed ./cmd/devseed

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=api-build /out/visionopus-api /app/visionopus-api
COPY --from=api-build /out/visionopus-migrate /app/visionopus-migrate
COPY --from=api-build /out/visionopus-devseed /app/visionopus-devseed
COPY --from=web-build /src/web/dist /app/web

ENV VISIONOPUS_ENV=production \
    VISIONOPUS_HTTP_ADDR=:10000 \
    VISIONOPUS_STATIC_DIR=/app/web
EXPOSE 10000
USER nonroot:nonroot
ENTRYPOINT ["/app/visionopus-api"]
