FROM golang:1.26-bookworm

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        iproute2 \
        iputils-ping \
        nftables \
        iperf3 \
        tcpdump \
        clang \
        llvm \
        libbpf-dev \
        gcc \
        make \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["go", "run", "./cmd/lab"]
