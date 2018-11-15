FROM alpine:3.8

ARG APP_NAME

COPY "${APP_NAME}" /usr/local/bin/slack-duty-bot

RUN apk --no-cache add ca-certificates && \
    chmod +x /usr/local/bin/slack-duty-bot

ENTRYPOINT ["slack-duty-bot"]
