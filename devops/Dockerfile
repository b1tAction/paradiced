FROM golang:1.22-bookworm

RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs git curl && \
    rm -rf /var/lib/apt/lists/*

RUN npm install -g @anthropic-ai/claude-code

ARG UID=1000
ARG GID=1000

RUN groupadd -g ${GID} claude && \
    useradd -u ${UID} -g ${GID} -m -s /bin/bash claude && \
    echo "claude ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

RUN mkdir -p /app /go/pkg && \
    chown -R claude:claude /app /go

USER claude
WORKDIR /app
ENV GOPATH=/go

CMD ["/bin/bash"]