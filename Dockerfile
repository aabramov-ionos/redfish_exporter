FROM golang:1.25-alpine AS builder

ARG ARCH=amd64

ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH="$GOROOT/bin:$GOPATH/bin:$PATH"
ENV GO_VERSION=1.25
ENV GO111MODULE=on

# Install git for dependencies
RUN apk add --no-cache git make

# Copy source code
COPY . /go/src/github.com/jenningsloy318/redfish_exporter
WORKDIR /go/src/github.com/jenningsloy318/redfish_exporter
RUN make build

FROM golang:1.25-alpine


COPY --from=builder /go/src/github.com/jenningsloy318/redfish_exporter/build/redfish_exporter /usr/local/bin/redfish_exporter
RUN mkdir /etc/prometheus
COPY config.yml.example /etc/prometheus/redfish_exporter.yml
CMD ["/usr/local/bin/redfish_exporter","--config.file","/etc/prometheus/redfish_exporter.yml"]
