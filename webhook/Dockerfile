# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o server \
    .

# Final stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup
WORKDIR /app

COPY --from=builder /build/server .
RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 8080

ENTRYPOINT [ "./server" ]