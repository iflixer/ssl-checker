# build environment
FROM golang:1.23 AS build-env
WORKDIR /server
COPY src/go.mod ./
RUN go mod download
COPY src src
WORKDIR /server/src
RUN CGO_ENABLED=0 GOOS=linux go build -o /server/build/build .

FROM alpine:3.21
WORKDIR /app
RUN mkdir tmp

COPY --from=build-env /server/build/build /app/ssl-checker


ENTRYPOINT [ "/app/ssl-checker" ]
