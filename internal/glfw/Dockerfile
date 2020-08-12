FROM debian:testing

RUN apt-get update && apt-get install -y \
        ca-certificates \
        golang \
        gcc-mingw-w64=10.1.0-3+23 \
        && rm -rf /var/lib/apt/lists/*

WORKDIR /work
