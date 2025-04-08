ARG GO_VERSION=1.23
FROM golang:${GO_VERSION}-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN ./scripts/package.sh


FROM golang:${GO_VERSION}-bookworm
WORKDIR /app

ENV DEBIAN_FRONTEND=noninteractive

COPY --from=builder /app/dist .

RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | gpg --dearmor -o /etc/apt/trusted.gpg.d/google.gpg \
    && echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list \
    && apt-get update -y \
    && apt install -y google-chrome-stable 
    
CMD ["./app"]
