FROM golang:1.22.2-bullseye AS build

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y libudev-dev

WORKDIR /app
RUN git clone https://github.com/jurkovic-nikola/OpenLinkHub.git

WORKDIR /app/OpenLinkHub
RUN go build .

FROM debian:bullseye-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y libudev-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
    
RUN mkdir -p /opt/OpenLinkHub

COPY --from=build /app/OpenLinkHub/OpenLinkHub /opt/OpenLinkHub/
COPY --from=build /app/OpenLinkHub/database /opt/OpenLinkHub/database
COPY --from=build /app/OpenLinkHub/static /opt/OpenLinkHub/static
COPY --from=build /app/OpenLinkHub/web /opt/OpenLinkHub/web

WORKDIR /opt/OpenLinkHub

ENTRYPOINT ["/opt/OpenLinkHub/OpenLinkHub"]
