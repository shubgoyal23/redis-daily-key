# --- build ---------------------------------------------------------------
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/redis-daily .

# --- runtime -------------------------------------------------------------
# busybox crond (part of alpine) schedules the job; tzdata lets TZ decide
# what "midnight" means.
FROM alpine:3.22
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /app
COPY --from=build /out/redis-daily /app/redis-daily
COPY docker/crontab /etc/crontabs/root
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
