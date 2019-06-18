FROM golang:1.12 AS builder
WORKDIR /app
COPY . .
RUN GOBIN=$(pwd)/bin CGO_ENABLED=0 GOOS=linux go install -a -installsuffix cgo ./cmd/...

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app/bin
COPY --from=builder /app/bin /app/bin
ENV PATH="/app/bin:${PATH}"
CMD echo "Available commands:" && ls
