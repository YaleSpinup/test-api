# build stage
FROM golang:alpine AS build-env
RUN apk add --no-cache git gcc musl-dev

RUN mkdir /app
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -trimpath -o /go/api *.go

# final stage
FROM alpine

WORKDIR /app
COPY --from=build-env /go/api /app/api
COPY --from=build-env /app/README.md /app/README.md

RUN chmod 555 /app/api && chown -R nobody:nogroup /app
CMD /app/api -debug