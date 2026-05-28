FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /product-service .

FROM alpine:3.18

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /product-service .

EXPOSE 8082

CMD ["./product-service"]
