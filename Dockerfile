FROM golang:1.15-alpine as builder

WORKDIR ${GOPATH}/src/github.com/bbaobelief/k8s-falcon-metrics-adapter
COPY . ./

# RUN CGO_ENABLED=0 go test $(go list ./... | grep -v -e '/client/' -e '/samples/' -e '/apis/')
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -tags=netgo -o /adapter main.go

FROM alpine:3.10
RUN apk update \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/* \
    && update-ca-certificates

ENTRYPOINT ["/adapter", "--logtostderr=true"]
COPY --from=builder /adapter /
# build
# docker build -t k8s-falcon-metrics-adapter .