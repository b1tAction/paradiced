#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
FORCE_REBUILD=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    echo "Usage: $0 [-f]"
    echo "  -f    Force rebuild: stop and remove existing container, then rebuild"
    exit 0
}

parse_args() {
    while getopts "fh" opt; do
        case $opt in
            f) FORCE_REBUILD=true ;;
            h) usage ;;
            ?) usage ;;
        esac
    done
}

# Check if container is already running
check_running_container() {
    if docker ps --format '{{.Names}}' | grep -q '^claude_backend_dev$'; then
        return 0
    fi
    return 1
}

# Check if container exists (stopped)
check_stopped_container() {
    if docker ps -a --format '{{.Names}}' | grep -q '^claude_backend_dev$'; then
        return 0
    fi
    return 1
}

# Setup .claude directory with correct permissions
setup_claude_directory() {
    local claude_dir="$PROJECT_ROOT/.claude"

    if [ -d "$claude_dir" ]; then
        # Directory exists, check if we need to fix permissions
        local current_owner=$(stat -c '%U' "$claude_dir" 2>/dev/null || stat -f '%Su' "$claude_dir" 2>/dev/null)
        if [ "$current_owner" = "root" ]; then
            log_info "Fixing .claude directory permissions..."
            sudo chown -R $(id -u):$(id -g) "$claude_dir"
            sudo chmod -R 777 "$claude_dir"
        fi
    else
        # Create directory with current user ownership
        log_info "Creating .claude directory..."
        mkdir -p "$claude_dir"
        chmod 777 "$claude_dir"
    fi
}

# Main setup logic
main() {
    parse_args "$@"

    log_info "Starting devops setup..."
    log_info "Project root: $PROJECT_ROOT"

    # Setup .claude directory first
    setup_claude_directory

    cd "$SCRIPT_DIR"

    # Force rebuild: down and rebuild
    if [ "$FORCE_REBUILD" = true ]; then
        log_info "Force rebuild requested..."
        if check_running_container || check_stopped_container; then
            log_info "Stopping and removing existing container..."
            docker compose down
        fi
        log_info "Building and starting container (UID=$(id -u), GID=$(id -g))..."
        USER_UID=$(id -u) USER_GID=$(id -g) docker compose up -d --build
        log_info "Setup complete!"
        log_info "To attach to the container, run: docker exec -it claude_backend_dev bash"
        log_info "To view logs, run: docker logs -f claude_backend_dev"
        exit 0
    fi

    if check_running_container; then
        log_info "Container 'claude_backend_dev' is already running."
        log_info "To attach to it, run: docker exec -it claude_backend_dev bash"
        log_info "To stop it, run: docker compose down"
        log_info "To force rebuild, run: $0 -f"
        exit 0
    fi

    if check_stopped_container; then
        log_info "Container 'claude_backend_dev' exists but is stopped. Starting it..."
        docker compose up -d
    else
        log_info "Building and starting container (UID=$(id -u), GID=$(id -g))..."
        USER_UID=$(id -u) USER_GID=$(id -g) docker compose up -d --build
    fi

    log_info "Setup complete!"
    log_info "To attach to the container, run: docker exec -it claude_backend_dev bash"
    log_info "To view logs, run: docker logs -f claude_backend_dev"
}

main "$@"