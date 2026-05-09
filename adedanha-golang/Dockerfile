# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -tags "linux" -ldflags="-s -w" -o /app/server .

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /app/server .

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./server"]
