FROM ghcr.io/formancehq/base:scratch
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/fctl /usr/bin/fctl
ENV OTEL_SERVICE_NAME fctl
ENTRYPOINT ["/usr/bin/fctl"]
