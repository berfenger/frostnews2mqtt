FROM alpine:3.20.1 AS prod

RUN apk add ca-certificates curl gcompat

WORKDIR /app
COPY bin/frostnews2mqtt /app/frostnews2mqtt
EXPOSE ${PORT:-8080}

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=5s CMD curl -f http://127.0.0.1:${PORT:-8080}/healthcheck || exit 1

ENTRYPOINT ["/app/frostnews2mqtt"]
