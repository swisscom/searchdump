FROM golang:1.17-alpine3.15 AS builder
COPY / /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -o /searchdump ./cmd/searchdump

FROM alpine:3.15
COPY --from=builder /searchdump /usr/bin/searchdump
ENTRYPOINT ["/usr/bin/searchdump"]