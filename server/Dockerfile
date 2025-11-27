# server/Dockerfile
FROM golang:1.25.4-alpine AS build

WORKDIR /app
COPY . .
RUN go build -o grid-api ./cmd/api

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/grid-api /app/grid-api

ENV REDIS_ADDR=redis:6379
ENV REDIS_PASSWORD=
EXPOSE 8080

CMD ["/app/grid-api"]
