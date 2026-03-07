FROM ghcr.io/hassio-addons/base:20.0.1

SHELL ["/bin/bash", "-lc"]

# Core tools and filesystem/test dependencies used by SRAT development workflows.
RUN apk add --no-cache \
    bash \
    btrfs-progs \
    curl \
    dosfstools \
    e2fsprogs \
    e2fsprogs-extra \
    ethtool \
    eudev \
    exfatprogs \
    f2fs-tools \
    fuse3 \
    gcc \
    git \
    git-bash-completion \
    git-prompt \
    graphviz \
    hfsprogs \
    jq \
    linux-headers \
    lsblk \
    make \
    minisign \
    musl-dev \
    nix \
    ntfs-3g-progs \
    openssh-client \
    patch \
    py3-mypy \
    py3-pip \
    py3-pytest \
    py3-ruff \
    python3 \
    ruff \
    samba \
    shadow \
    smartmontools \
    sshfs \
    wget \
    xfsprogs-extra \
    zig \
    apfs-fuse \
    act

# Documentation validation tools (from edge repositories).
RUN apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community vale \
    && apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/testing lychee

# Install Go from go.dev/dl based on backend/src/go.mod.
COPY backend/src/go.mod /tmp/srat-go.mod
RUN GO_VERSION="$(awk '/^go / {print $2}' /tmp/srat-go.mod)" \
    && ARCH_RAW="$(uname -m)" \
    && case "$ARCH_RAW" in \
    x86_64) GO_ARCH="amd64" ;; \
    aarch64) GO_ARCH="arm64" ;; \
    *) echo "Unsupported architecture: ${ARCH_RAW}"; exit 1 ;; \
    esac \
    && wget -nv "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm -f /tmp/go.tar.gz \
    && ln -sf /usr/local/go/bin/go /usr/local/bin/go \
    && ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Install Bun based on frontend/package.json packageManager.
COPY frontend/package.json /tmp/srat-package.json
ENV BUN_INSTALL=/root/.bun
ENV PATH=/root/.bun/bin:${PATH}
RUN BUN_VERSION="$(jq -r '.packageManager' /tmp/srat-package.json | sed 's/bun@/bun-v/')" \
    && curl -fsSL https://bun.sh/install | bash -s "$BUN_VERSION" \
    && ln -sf /root/.bun/bin/bun /root/.bun/bin/bunx 

# Optional global tooling used by contributor workflows.
RUN PATH=/root/.bun/bin:${PATH} \
    && bun install -g @j178/prek \
    && bun add -g biome @biomejs/cli-linux-arm64-musl \
    && bun pm -g trust --all
