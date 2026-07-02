FROM mcr.microsoft.com/devcontainers/go:2-1.26-bookworm@sha256:090a56c6c9c4e80a1573c18c1225eb851b8c86c199fc929b07ea2a67f7b4850f

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
RUN #--mount=type=secret,id=github_token,env=GITHUB_TOKEN \
    echo 'eval "$(mise activate zsh --shims)"' >> /root/.zshrc \
    && echo 'eval "$(mise activate bash --shims)"' >> /root/.bashrc
