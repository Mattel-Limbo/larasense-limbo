# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags "-X github.com/Mattel-Limbo/larasense-limbo/cmd.Version=${VERSION} \
              -X github.com/Mattel-Limbo/larasense-limbo/cmd.Commit=${COMMIT} \
              -X github.com/Mattel-Limbo/larasense-limbo/cmd.BuildDate=${BUILD_DATE}" \
    -o /bin/larasense-limbo .

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache git

COPY --from=builder /bin/larasense-limbo /usr/local/bin/larasense-limbo

WORKDIR /repo

ENTRYPOINT ["larasense-limbo"]
CMD ["analyze"]
