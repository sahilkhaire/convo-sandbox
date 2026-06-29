FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /server /app/server
RUN mkdir -p /data/media
VOLUME /data/media
ENV PORT=8080
ENV MEDIA_STORAGE_PATH=/data/media
EXPOSE 8080
CMD ["/app/server"]
