# Build the manager binary
FROM golang:1.17 as builder

RUN apt update && apt install -y curl gnupg2

RUN curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc > /tmp/pgsql.gpg
RUN apt-key add /tmp/pgsql.gpg
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ buster-pgdg main" > /etc/apt/sources.list.d/pgdg.list
RUN apt-get update && apt-get install -y postgresql-client-12 jq

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY app/cmd/ app/cmd/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/app app/cmd/produce.go

COPY images/helloproduce/entrypoint.sh /entrypoint.sh
RUN chmod a+x /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]

FROM rabbitmq:3.9.13-management

# Update system (clear security vulns)
RUN apt-get -y -qq update \
 && apt-get -y -qq upgrade \
 && apt-get -y -qq clean \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Prometheus: https://github.com/rabbitmq/rabbitmq-prometheus
# K8s peer discovery: https://github.com/rabbitmq/rabbitmq-peer-discovery-k8s
# Management: https://github.com/rabbitmq/rabbitmq-management
# HTTP Auth: https://github.com/rabbitmq/rabbitmq-auth-backend-http
# Cache auth: https://github.com/rabbitmq/rabbitmq-auth-backend-cache

RUN rabbitmq-plugins --offline enable rabbitmq_peer_discovery_k8s \
 && rabbitmq-plugins --offline enable rabbitmq_prometheus \
 && rabbitmq-plugins --offline enable rabbitmq_management \
 && rabbitmq-plugins --offline enable rabbitmq_auth_backend_http \
 && rabbitmq-plugins --offline enable rabbitmq_auth_backend_cache \
 && rabbitmq-plugins --offline enable rabbitmq_federation \
 && rabbitmq-plugins --offline enable rabbitmq_federation_management

COPY --from=builder /entrypoint.sh /entrypoint.sh
COPY --from=builder /workspace/bin/app /workspace/bin/app

ENTRYPOINT ["/bin/bash", "/entrypoint.sh"]


# # Use distroless as minimal base image to package the manager binary
# # Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/static:nonroot
# WORKDIR /
# COPY --from=builder /workspace/bin/app .
# USER nonroot:nonroot

# ENTRYPOINT ["/app"]
