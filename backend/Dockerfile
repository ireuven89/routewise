FROM golang:1.25-alpine3.23 AS builder


WORKDIR APP

COPY go.mod ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /backend ./cmd/server


FROM alpine:3.17


WORKDIR /root/

COPY --from=builder /backend .


CMD ["./backend"]
