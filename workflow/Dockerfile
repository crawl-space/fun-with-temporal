FROM golang:1.16.3

WORKDIR /go/src/app
COPY go.mod .
COPY go.sum .

RUN go get -d -v ./...

EXPOSE 6007

COPY . .

RUN go install -v ./...

CMD ["workflow"]