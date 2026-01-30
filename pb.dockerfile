# --== build image ==--
FROM golang:alpine AS builder

# Install gcc and musl-dev for CGO support
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum /build/

RUN go mod download

COPY . /build

# Build with CGO enabled for SQLite support
RUN CGO_ENABLED=1 go build -o pb

#--== final image ==--
FROM alpine

COPY --from=builder /build/pb /opt/pb/

WORKDIR /opt/pb

CMD ["./pb"]
