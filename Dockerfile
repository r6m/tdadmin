FROM golang:1.12-alpine3.9 as builder
RUN apk add --no-cache --virtual .build-deps linux-headers git gcc g++ make cmake gperf libc-dev openssl-dev zlib-dev tzdata ca-certificates git curl gcc upx

WORKDIR src/tdadmin
COPY --from=zenwalker/tdlib /usr/local/include/td /usr/local/include/td
COPY --from=zenwalker/tdlib /usr/local/lib/libtd* /usr/local/lib/

COPY . .
RUN curl https://glide.sh/get | sh && glide i
RUN go build -o /app/app main.go
RUN upx /app/app

FROM alpine:3.10
RUN apk add --no-cache tzdata ca-certificates libstdc++ libssl1.1 libcrypto1.1

WORKDIR /app
COPY --from=zenwalker/tdlib /usr/local/include/td /usr/local/include/td
COPY --from=zenwalker/tdlib /usr/local/lib/libtd* /usr/local/lib/
COPY --from=builder  /app/app /app/app

ENTRYPOINT [ "/app/app" ]