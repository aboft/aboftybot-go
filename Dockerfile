FROM golang:1.26-alpine

RUN mkdir /aboftybot
ADD . /aboftybot
WORKDIR /aboftybot

RUN env GOOS=linux GOARCH=amd64 go build -o aboftybot .
RUN apk add --no-cache tzdata

ENV TZ=America/New_York
ENV GOPATH=$PATH:/

CMD [ "/aboftybot/aboftybot" ] 
