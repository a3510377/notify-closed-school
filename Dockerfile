FROM golang:alpine as builder

WORKDIR /app
COPY go.* ./
RUN go mod download
COPY *.go ./
RUN go build -v -a -ldflags '-s -w' -gcflags="all=-trimpath=${PWD}" -asmflags="all=-trimpath=${PWD}" -o start

FROM alpine
WORKDIR /app
COPY --from=builder /app/start .
RUN apk add tzdata

VOLUME [ "/app/data" ]

CMD /app/start
