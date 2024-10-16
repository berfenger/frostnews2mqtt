FROM golang:1.23-alpine AS build

RUN apk add ca-certificates make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

#RUN go build -o frostnews2mqtt cmd/api/main.go
RUN make build

FROM alpine:3.20.1 AS prod

RUN apk add ca-certificates curl

WORKDIR /app
COPY --from=build /app/frostnews2mqtt /app/frostnews2mqtt
EXPOSE ${FROSTNEWS_PORT}

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=5s CMD curl -f http://127.0.0.1:${FROSTNEWS_PORT:-8080}/ || exit 1

CMD ["./frostnews2mqtt"]
