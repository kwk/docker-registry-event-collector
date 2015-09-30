# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.4

MAINTAINER "Konrad Kleine"

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/kwk/docker-registry-event-collector

# Fetch all dependencies
RUN cd /go/src/github.com/kwk/docker-registry-event-collector \
    && go get

# Build the docker-registry-event-collector command inside the container.
RUN go install github.com/kwk/docker-registry-event-collector

# Run the docker-registry-event-collector command by default when the
# container starts.
ENTRYPOINT /go/bin/docker-registry-event-collector
