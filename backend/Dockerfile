FROM golang:1.24.13-alpine3.22 AS build
RUN --mount=type=secret,id=extra_ca if [ -f /run/secrets/extra_ca ]; then cat /run/secrets/extra_ca >> /etc/ssl/certs/ca-certificates.crt; fi
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -o /server ./cmd/server
FROM alpine:3.21
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN adduser -D app
USER app
COPY --from=build /server /server
EXPOSE 8080
CMD ["/server"]
