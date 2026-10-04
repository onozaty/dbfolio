FROM mcr.microsoft.com/devcontainers/go:1.27-bookworm

# Install PostgreSQL 18 client from the official PGDG repository to match the server version
RUN apt-get update \
    && apt-get install -y --no-install-recommends postgresql-common \
    && /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y \
    && apt-get install -y --no-install-recommends postgresql-client-18 \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /home/vscode/.claude /home/vscode/.config/gh && chown -R vscode:vscode /home/vscode/.claude /home/vscode/.config/gh
