FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.24 As builder

ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ENV VERSION=$VERSION

WORKDIR /app/
ADD . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-X main.version=$VERSION" \
    -o k8s-restarter \
    k8s-restarter.go

FROM alpine

RUN apk add --no-cache tzdata

WORKDIR /app
COPY --from=builder /app/k8s-restarter /app/k8s-restarter

RUN /usr/sbin/addgroup app
RUN /usr/sbin/adduser app -G app -D
USER app

ENTRYPOINT ["/app/k8s-restarter"]
CMD []
