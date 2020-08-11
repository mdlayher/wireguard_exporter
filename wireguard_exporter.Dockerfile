# build
FROM golang:latest AS build
WORKDIR /build

ADD . /build
RUN cd cmd/wireguard_exporter/ && \
    go build . && \
    mv wireguard_exporter /usr/local/bin/

# image
FROM ubuntu:20.04
WORKDIR /usr/src/app/

# update image
RUN apt-get update && \
    apt-get dist-upgrade -y && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /usr/local/bin/wireguard_exporter /usr/local/bin/

EXPOSE 9586
ENTRYPOINT ["/usr/local/bin/wireguard_exporter", "-wireguard.peer-file", "/usr/src/app/peers.toml"]
