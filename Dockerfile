
FROM golang:1.23

WORKDIR /app

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./src

EXPOSE 10823/tcp
EXPOSE 10823/udp

CMD ["/server"]
