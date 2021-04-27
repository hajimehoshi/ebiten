FROM debian:testing

# For the version of gcc-mingw-w64, see https://packages.debian.org/bullseye/gcc-mingw-w64-x86-64
RUN apt-get update && apt-get install -y \
        ca-certificates \
        golang \
        gcc-mingw-w64=10.2.1-6+24.1 \
        && rm -rf /var/lib/apt/lists/*

WORKDIR /work
