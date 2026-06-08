FROM golang:1.26-bookworm AS base

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o linky-winky
EXPOSE 8080
CMD ["/build/linky-winky"]
