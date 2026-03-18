FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /build/server .

EXPOSE 2757

ENV SERVER_PORT=2757

CMD ["./server"]
