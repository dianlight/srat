FROM mcr.microsoft.com/devcontainers/go:2-1.26-bookworm@sha256:fe5d30ee051db0b5bd7818f3861a4d60a8a5f1b937a8a0c0fe14f6dd656f962f

# Filesystem, SAMBA, and block-device tools needed for SRAT development/testing.
RUN apt-get update && apt-get install -y --no-install-recommends \
    btrfs-progs \
    dosfstools \
    e2fsprogs \
    ethtool \
    exfatprogs \
    f2fs-tools \
    fuse3 \
    graphviz \
#    hfsprogs \
#    linux-libc-dev \
    minisign \
    ntfs-3g \
    patch \
    samba \
    smartmontools \
    sshfs \
    xfsprogs \
#    zfsutils-linux \
#    libfuse3-dev \
#    libbz2-dev \
#    libattr1-dev \
#    zlib1g-dev \
#    cmake \
#    g++ \
    && rm -rf /var/lib/apt/lists/*

# Build apfs-fuse from source (not packaged in Debian/Ubuntu).
# RUN git clone --depth 1 https://github.com/sber-devops/apfs-fuse.git /tmp/apfs-fuse \
#     && cd /tmp/apfs-fuse && git submodule update --init --recursive \
#     && mkdir build && cd build \
#     && cmake .. -DCMAKE_INSTALL_PREFIX=/usr/local \
#     && make -j"$(nproc)" && make install \
#     && rm -rf /tmp/apfs-fuse

# Remove build-only deps to keep image lean.
# RUN apt-get purge -y cmake g++ libfuse3-dev libbz2-dev libattr1-dev zlib1g-dev \
#     && apt-get autoremove -y && rm -rf /var/lib/apt/lists/*

# Install mise (manages Go, Bun, Python, and all other pinned tool versions).
# COPY .mise.toml /tmp/.mise.toml
RUN --mount=type=secret,id=github_token,env=GITHUB_TOKEN \
#     curl -fsSL https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise sh \
     && echo 'eval "$(mise activate zsh --shims)"' >> /root/.zshrc \
     && echo 'eval "$(mise activate bash --shims)"' >> /root/.bashrc
#     && cd /tmp \
#     && MISE_EXPERIMENTAL=1 mise install --yes --verbose
