FROM golang:1.16 as base

COPY ./ /app

WORKDIR /app/go_app

RUN go mod download && go get github.com/githubnemo/CompileDaemon

ENTRYPOINT CompileDaemon --build="go build server.go" --command=./server