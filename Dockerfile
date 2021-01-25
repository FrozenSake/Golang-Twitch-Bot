FROM golang:1.14 AS builder

WORKDIR /go/src/chatbot
COPY ./app .

RUN go get -d -v ./...
RUN go install -v ./...

FROM golang:1.14 as runner

WORKDIR /go/src/chatbot
COPY --from=builder /go/src/twitchbot/golang-twitch-bot .

CMD ["golang-twitch-bot"]