FROM alpine:3.20.1 AS prod

RUN apk add ca-certificates curl

WORKDIR /app
COPY frostnews2mqtt /app/frostnews2mqtt
EXPOSE ${FROSTNEWS_PORT}

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=5s CMD curl -f http://127.0.0.1:${FROSTNEWS_PORT:-8080}/ || exit 1

CMD ["/app/frostnews2mqtt"]
