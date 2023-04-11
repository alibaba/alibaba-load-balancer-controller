FROM alpine:3.11.6

# Do not use docker multiple stage build until we
# figure a way out how to solve build cache problem under 'go mod'.
#

RUN apk add --no-cache --update ca-certificates

COPY bin/load-balancer-controller /load-balancer-controller

ENTRYPOINT  ["/load-balancer-controller"]