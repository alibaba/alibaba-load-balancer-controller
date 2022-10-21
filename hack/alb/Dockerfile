FROM golang:1.17.6

WORKDIR /alibaba-load-balancer-controller

ENV GOPROXY=https://goproxy.cn

COPY cmd /alibaba-load-balancer-controller/cmd
COPY version /alibaba-load-balancer-controller/version
COPY Makefile /alibaba-load-balancer-controller/Makefile
COPY go.mod /alibaba-load-balancer-controller/go.mod
COPY go.sum /alibaba-load-balancer-controller/go.sum
COPY vendor /alibaba-load-balancer-controller/vendor
COPY hack /alibaba-load-balancer-controller/hack
COPY pkg /alibaba-load-balancer-controller/pkg
COPY test /alibaba-load-balancer-controller/test
# RUN cd /alibaba-load-balancer-controller/test/e2e && go test ./ -c -o alb.test