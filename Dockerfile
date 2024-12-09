ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /run-app .


FROM debian:bookworm
RUN apt update \
        && apt install --yes ca-certificates \
        && update-ca-certificates 2>/dev/null

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app"]
