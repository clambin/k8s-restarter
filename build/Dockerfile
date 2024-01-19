FROM alpine

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

COPY bld/k8s-restarter-$TARGETOS-$TARGETARCH /app/k8s-restarter

EXPOSE 9091
ENTRYPOINT ["/app/k8s-restarter"]
CMD []
