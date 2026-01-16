FROM golang:1.23.8-bullseye AS build
ARG GIT_TAG
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y libudev-dev i2c-tools libpipewire-0.3-dev pkg-config
RUN mkdir -p /opt/OpenLinkHub

WORKDIR /app
COPY . /app/OpenLinkHub

WORKDIR /app/OpenLinkHub
RUN if [ -n "$GIT_TAG" ]; then git checkout "$GIT_TAG"; fi
RUN go build .

FROM debian:bullseye-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y libudev-dev pciutils usbutils udev i2c-tools && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc/modules-load.d
RUN echo 'KERNEL=="i2c-0", MODE="0600", OWNER="openlinkhub"' | tee /etc/udev/rules.d/98-corsair-memory.rules
RUN echo "i2c-dev" | tee /etc/modules-load.d/i2c-dev.conf

COPY --from=build /app/OpenLinkHub/OpenLinkHub /opt/OpenLinkHub/
COPY --from=build /app/OpenLinkHub/database /opt/OpenLinkHub/database
COPY --from=build /app/OpenLinkHub/static /opt/OpenLinkHub/static
COPY --from=build /app/OpenLinkHub/web /opt/OpenLinkHub/web
COPY --from=build /app/OpenLinkHub/99-openlinkhub.rules /etc/udev/rules.d/99-openlinkhub.rules

WORKDIR /opt/OpenLinkHub

ENTRYPOINT ["/opt/OpenLinkHub/OpenLinkHub"]