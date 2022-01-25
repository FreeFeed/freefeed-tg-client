# Based on recipie described here:
# https://chemidy.medium.com/create-the-smallest-and-secured-golang-docker-image-based-on-scratch-4752223b7324

# BUILDER IMAGE
############################
# FROM golang:golang:1.17.6-alpine3.15 AS builder
FROM golang@sha256:f28579af8a31c28fc180fb2e26c415bf6211a21fb9f3ed5e81bcdbf062c52893 as builder

# Git is required for fetching the dependencies
RUN apk update && apk add --no-cache git

WORKDIR /app
COPY . .

RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o freefeed-tg-client

# PRODUCTION IMAGE
############################
# FROM alpine:3.15.0
FROM alpine@sha256:e7d88de73db3d3fd9b2d63aa7f447a10fd0220b7cbf39803c803f2af9ba256b3        

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
