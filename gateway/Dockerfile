FROM golang:1.22-alpine AS build
WORKDIR /src
COPY gateway/ .
RUN CGO_ENABLED=0 go build -o /gateway .

FROM alpine:3.20
COPY --from=build /gateway /gateway
COPY web /web
ENV WEB_DIR=/web
EXPOSE 8080
CMD ["/gateway"]
