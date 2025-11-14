# Based on Microsoft's devcontainer base image with batteries-included agentic development tools
# Optimized for daily autobuilds with proper layer caching
FROM mcr.microsoft.com/devcontainers/base:ubuntu

# Layer 1: Base system dependencies and utilities (most stable)
RUN apt-get update && \
    apt-get install -y \
        curl wget unzip \
        jq yq \
        vim nano \
        make \
        build-essential \
        python3 python3-pip && \
    # Cleanup
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Layer 2: Install uv (modern Python package manager)
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

# Layer 3: Install Go
RUN GO_VERSION="1.21.5" && \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -xzC /usr/local && \
    echo 'export PATH=/usr/local/go/bin:$PATH' >> /etc/profile

# Layer 4: Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && \
    echo 'source ~/.cargo/env' >> /etc/profile

# Layer 5: Install Node.js LTS
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - && \
    apt-get install -y nodejs

# Layer 6: Install cloud CLIs (moderately stable)
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then \
        curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"; \
    elif [ "$ARCH" = "arm64" ]; then \
        curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o "awscliv2.zip"; \
    else \
        echo "Unsupported architecture: $ARCH"; exit 1; \
    fi && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -rf aws awscliv2.zip && \
    # Install Azure CLI
    curl -sL https://aka.ms/InstallAzureCLIDeb | bash && \
    # Install Google Cloud CLI
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    apt-get update && apt-get install -y google-cloud-cli && \
    # Install GitHub CLI
    mkdir -p -m 755 /etc/apt/keyrings && \
    wget -nv -O- https://cli.github.com/packages/githubcli-archive-keyring.gpg | tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null && \
    chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && apt-get install -y gh && \
    # Cleanup
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Layer 7: AI CLI tools (most volatile - frequent releases)
# Install in order of stability (most stable first for better caching)
RUN npm install -g \
        @sourcegraph/amp \
        @qwen-code/qwen-code \
        @google/gemini-cli \
        @github/copilot \
        @openai/codex \
        @anthropic-ai/claude-code

# Layer 8: Final environment setup
ENV PATH="/usr/local/go/bin:/home/vscode/.cargo/bin:${PATH}"
ENV DISABLE_AUTOUPDATER=true

# Set the default user (matches devcontainer convention)
USER vscode
