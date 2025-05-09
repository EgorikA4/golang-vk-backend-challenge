FROM golang:1.24.2-alpine3.21 AS builder

WORKDIR /code

COPY ./cmd ./cmd
COPY ./go.mod ./go.sum ./
COPY ./internal ./internal

RUN go build -o app ./cmd/main.go

FROM alpine:3.21

WORKDIR /myapp

COPY --from=builder /code/app ./

EXPOSE 8001

CMD [ "/myapp/app" ]
