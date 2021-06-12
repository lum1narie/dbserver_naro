FROM golang:1.16-alpine
RUN apk add --update --no-cache ca-certificates git

WORKDIR /work
COPY . .
RUN go build -o app
ENTRYPOINT ./app