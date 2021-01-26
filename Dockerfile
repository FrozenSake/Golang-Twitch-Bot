FROM golang:1.14 AS getter

WORKDIR /go/src/chatbot
COPY ./app .

RUN go get -d -v ./...

FROM getter AS builder
WORKDIR /go/src/chatbot

RUN go install -v ./...

FROM golang:1.14 as runner

WORKDIR /go/bin/
COPY --from=builder /go/bin/golang-twitch-bot .

CMD ["/go/bin/golang-twitch-bot"]