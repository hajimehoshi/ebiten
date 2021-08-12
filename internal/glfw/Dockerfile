FROM debian:buster

# For the version of gcc-mingw-w64, see https://packages.debian.org/buster/gcc-mingw-w64-x86-64
RUN apt-get update && apt-get install -y \
        git \
        ca-certificates \
        golang \
        gcc-mingw-w64=8.3.0-6+21.3~deb10u1 \
        && rm -rf /var/lib/apt/lists/*

WORKDIR /work
