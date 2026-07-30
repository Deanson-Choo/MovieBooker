# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy the full backend source.
COPY . .

RUN go build -o server ./api/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/server .

ENV GIN_MODE=release

CMD ["./server"]