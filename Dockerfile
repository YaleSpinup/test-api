# build stage
FROM golang:alpine AS build-env

ARG version="0.0.0"
ARG githash=""
ARG buildstamp=""

RUN apk add --no-cache git gcc musl-dev

RUN mkdir /app
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -trimpath -o /go/api -ldflags="-X main.Version=$version -X main.Githash=$githash -X main.Buildstamp=$buildstamp" *.go

# final stage
FROM alpine

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
  CMD wget -qO- localhost:8080/v1/test/ping || exit 1

WORKDIR /app
COPY --from=build-env /go/api /app/api
COPY --from=build-env /app/README.md /app/README.md

RUN chmod 555 /app/api && chown -R nobody:nogroup /app
CMD /app/api -debug