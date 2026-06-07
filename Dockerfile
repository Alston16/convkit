FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /bin/server /bin/server
COPY config/ config/
COPY migrations/ migrations/
EXPOSE 8080
ENTRYPOINT ["/bin/server"]
