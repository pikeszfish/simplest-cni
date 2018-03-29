FROM golang as builder

COPY . /go/src/github.com/pikeszfish/simplest-cni/
WORKDIR /go/src/github.com/pikeszfish/simplest-cni/

RUN mkdir -p /dist
COPY k8s-install/bridge /dist
COPY k8s-install/host-local /dist
COPY k8s-install/loopback /dist
COPY k8s-install/portmap /dist
COPY k8s-install/scripts/install-cni.sh /dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-X main.buildstamp=`date '+%Y-%m-%d_%I:%M:%S'`" \
        -a -v -o /dist/simplest-cni github.com/pikeszfish/simplest-cni

######

FROM alpine:latest

COPY --from=builder /dist/bridge /opt/cni/bin/
COPY --from=builder /dist/host-local /opt/cni/bin/
COPY --from=builder /dist/loopback /opt/cni/bin/
COPY --from=builder /dist/portmap /opt/cni/bin/

RUN apk add --update bash

COPY --from=builder /dist/simplest-cni /
COPY --from=builder /dist/install-cni.sh /
