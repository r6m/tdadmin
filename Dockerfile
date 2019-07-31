FROM alpine:3.7 as builder


FROM alpine:latest
RUN apk --no-cache add tzdata ca-certificates
WORKDIR /app
COPY --from=builder ./bin/app /app/
ENTRYPOINT [ "/app/app" ]