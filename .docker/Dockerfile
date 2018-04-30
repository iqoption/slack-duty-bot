FROM alpine:3.7

ARG APP

COPY "${APP}" /usr/local/bin/slack-duty-bot
COPY .docker/docker-entrypoint /usr/local/bin/

RUN apk --no-cache add ca-certificates

ENTRYPOINT ["docker-entrypoint"]

CMD ["slack-duty-bot"]
