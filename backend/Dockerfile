FROM golang:1.26-alpine

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

# Copy the full backend source.
COPY . .

ENV GIN_MODE=debug

CMD ["go", "run", "./api/main.go"]