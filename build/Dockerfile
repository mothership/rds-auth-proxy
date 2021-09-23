# syntax = docker/dockerfile:1.0-experimental
FROM golang:1.16.0 as builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates gawk

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rds-auth-proxy .

COPY build/bin/import_certs.sh /bin/import_certs.sh
RUN /bin/import_certs.sh

FROM scratch as runtime
ENV PATH=/bin

COPY --from=builder /usr/src/app/rds-auth-proxy /bin/rds-auth-proxy
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs/

ENTRYPOINT ["/bin/rds-auth-proxy"]
CMD ["server"]
