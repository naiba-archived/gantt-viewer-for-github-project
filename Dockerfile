FROM golang:alpine AS binarybuilder
RUN apk --no-cache --no-progress add \
    gcc git musl-dev
WORKDIR /gantt
COPY . .
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && \
    go build -o app -ldflags="-s -w -X github.com/naiba/gantt-viewer-for-github-project/singleton.Version=${GIT_COMMIT:0:7}"

FROM alpine:latest
ENV TZ="Asia/Shanghai"
RUN apk --no-cache --no-progress add \
    ca-certificates \
    tzdata && \
    cp "/usr/share/zoneinfo/$TZ" /etc/localtime && \
    echo "$TZ" >  /etc/timezone
WORKDIR /gantt
COPY ./static ./static
COPY ./views ./views
COPY --from=binarybuilder /gantt/app ./app

VOLUME ["/gantt/data"]
EXPOSE 80
CMD ["/gantt/app"]
