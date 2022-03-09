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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/app app/cmd/main.go

COPY images/hellosvc/entrypoint.sh /entrypoint.sh
RUN chmod a+x /entrypoint.sh

RUN go install github.com/tsenart/vegeta@v12.7.0

ENTRYPOINT [ "/entrypoint.sh" ]

# # Use distroless as minimal base image to package the manager binary
# # Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/static:nonroot
# WORKDIR /
# COPY --from=builder /workspace/bin/app .
# USER nonroot:nonroot

# ENTRYPOINT ["/app"]
