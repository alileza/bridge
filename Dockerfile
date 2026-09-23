FROM golang:1.27-alpine AS gobuild

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /app/bridge

# final image
FROM alpine:3.24

WORKDIR /app

COPY --from=gobuild /app/bridge /bin/bridge

ENTRYPOINT [ "bridge" ]
