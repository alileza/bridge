FROM golang:1.22-alpine as gobuild

COPY . /app
WORKDIR /app
RUN go build -o /app/bridge

# final image
FROM alpine:3.17.3

WORKDIR /app

COPY --from=gobuild /app/bridge /bin/bridge

ENTRYPOINT [ "bridge" ]