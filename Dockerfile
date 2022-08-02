# Based on recipie described here:
# https://chemidy.medium.com/create-the-smallest-and-secured-golang-docker-image-based-on-scratch-4752223b7324

# BUILDER IMAGE
############################
FROM golang:1.18.5-alpine3.16 AS builder

# Git is required for fetching the dependencies
RUN apk update && apk add --no-cache git

WORKDIR /app
COPY . .

RUN go get -d -v
RUN GOOS=linux go build -ldflags="-w -s" -o freefeed-tg-client

# PRODUCTION IMAGE
############################
FROM alpine:3.16.0

WORKDIR /bot

VOLUME /bot/data

ENV UID 10001
ENV GID 10001
ENV TOKEN ""
ENV DEBUG ""


# Create unprivileged user
RUN addgroup -g "${GID}" bot && \
  adduser -g "" -s /bin/false -G bot -D -H -u "${UID}" bot && \
  mkdir -p data && \
  chown bot:bot data


# Copy our static executable
COPY --from=builder /app/freefeed-tg-client .
# Use an unprivileged user
USER bot:bot
# Run the app binary
ENTRYPOINT ./freefeed-tg-client -token "${TOKEN}" -debug "${DEBUG}"
