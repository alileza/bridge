FROM golang:1.22-alpine as gobuild

COPY . /app
WORKDIR /app
RUN go build -o /app/bridge

# frontend build
FROM node:21-alpine as reactbuild

WORKDIR /app
RUN npm install -g pnpm
COPY ./portal/package*.json ./
RUN pnpm install
COPY ./portal .
RUN pnpm build


# final image
FROM alpine:3.17.3

WORKDIR /app

COPY --from=gobuild /app/bridge /bin/bridge
COPY --from=reactbuild /app/dist /app/portal/dist

ENTRYPOINT [ "bridge" ]