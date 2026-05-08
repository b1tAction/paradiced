#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

ROOT_DIR="/opt/paradiced"
INCOMING_DIR="${ROOT_DIR}/incoming"
RELEASES_DIR="${ROOT_DIR}/releases"
CURRENT_DIR="${ROOT_DIR}/current"
STAGING_ROOT="${ROOT_DIR}/staging"
SHARED_DIR="${ROOT_DIR}/shared"
SHARED_RUNTIME_DIR="${ROOT_DIR}/shared/runtime"
LOCK_FILE="${ROOT_DIR}/deploy.lock"
RELEASE_MARKER=".deploy-release"

RUNTIME_DIRS=(
  "modules"
  "logs"
  ".gomodcache"
  ".gocache"
  ".gopath"
)

RSYNC_EXCLUDES=(
  --exclude='/.git'
  --exclude='/.git/'
  --exclude='/modules/'
  --exclude='/logs/'
  --exclude='/build/'
  --exclude='/.gomodcache/'
  --exclude='/.gocache/'
  --exclude='/.gopath/'
  --exclude='/.cache/'
  --exclude='/tmp/'
)

RSYNC_COPY_OPTIONS=(
  -rt
  --no-owner
  --no-group
  --no-perms
  --chmod=Du=rwx,Dgo=rx,Fu=rw,Fgo=r
)

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[deploy] %s\n' "$*"
}

archive_path="${1:-}"
release_id="${2:-}"
staging_dir=""
previous_release=""
rollback_in_progress=0
current_mutated=0
runtime_preserved=0

if [[ -z "${archive_path}" ]]; then
  die "usage: $0 <source-archive.tar.gz> [release-id]"
fi

if [[ -z "${release_id}" ]]; then
  archive_name="$(basename -- "${archive_path}")"
  release_id="${archive_name%.tar.gz}"
  release_id="${release_id%.tgz}"
  release_id="${release_id%.tar}"
fi

if [[ ! "${release_id}" =~ ^[A-Za-z0-9._-]{7,128}$ ]]; then
  die "release id must be 7-128 characters and contain only letters, numbers, dot, underscore, or dash"
fi

if [[ ! -f "${archive_path}" ]]; then
  die "archive does not exist: ${archive_path}"
fi

if [[ ! -r "${archive_path}" ]]; then
  die "archive is not readable: ${archive_path}"
fi

if [[ ! -s "${archive_path}" ]]; then
  die "archive is empty: ${archive_path}"
fi
if [[ -L "${archive_path}" ]]; then
  die "archive must be a regular file, not a symlink: ${archive_path}"
fi

for required_command in docker find flock rsync tar; do
  if ! command -v "${required_command}" >/dev/null; then
    die "required command not found: ${required_command}"
  fi
done

ensure_current_real_directory() {
  if [[ -L "${CURRENT_DIR}" ]]; then
    die "${CURRENT_DIR} must be a real directory, not a symlink"
  fi
  if [[ -e "${CURRENT_DIR}" && ! -d "${CURRENT_DIR}" ]]; then
    die "${CURRENT_DIR} exists but is not a directory"
  fi
}

ensure_directory_path_safe() {
  local path="$1"

  if [[ -L "${path}" ]]; then
    die "${path} must be a real directory, not a symlink"
  fi
  if [[ -e "${path}" && ! -d "${path}" ]]; then
    die "${path} exists but is not a directory"
  fi
}

ensure_file_path_safe() {
  local path="$1"

  if [[ -L "${path}" ]]; then
    die "${path} must be a regular file, not a symlink"
  fi
  if [[ -e "${path}" && ! -f "${path}" ]]; then
    die "${path} exists but is not a regular file"
  fi
}

ensure_shared_runtime_root_safe() {
  ensure_directory_path_safe "${SHARED_DIR}"
  ensure_directory_path_safe "${SHARED_RUNTIME_DIR}"
}

ensure_runtime_dir_safe() {
  local runtime_dir="$1"

  ensure_directory_path_safe "${CURRENT_DIR}/${runtime_dir}"
  ensure_directory_path_safe "${SHARED_RUNTIME_DIR}/${runtime_dir}"
}

ensure_directory_path_safe "${ROOT_DIR}"
mkdir -p "${ROOT_DIR}"
ensure_directory_path_safe "${ROOT_DIR}"
ensure_file_path_safe "${LOCK_FILE}"
exec 9>"${LOCK_FILE}"
if ! flock -n 9; then
  die "another deployment is already running"
fi

ensure_directory_path_safe "${ROOT_DIR}"
ensure_directory_path_safe "${INCOMING_DIR}"
ensure_directory_path_safe "${RELEASES_DIR}"
ensure_directory_path_safe "${STAGING_ROOT}"
ensure_shared_runtime_root_safe
mkdir -p "${INCOMING_DIR}" "${RELEASES_DIR}" "${STAGING_ROOT}" "${SHARED_DIR}" "${SHARED_RUNTIME_DIR}"
ensure_shared_runtime_root_safe
ensure_current_real_directory
mkdir -p "${CURRENT_DIR}"
chown root:root "${CURRENT_DIR}"
chmod 755 "${CURRENT_DIR}"

cleanup() {
  if [[ -n "${staging_dir}" && -d "${staging_dir}" ]]; then
    rm -rf -- "${staging_dir}"
  fi
}
trap cleanup EXIT

validate_archive_entries() {
  local entry
  local metadata
  local entry_type

  while IFS= read -r entry; do
    case "${entry}" in
      /*|../*|*/../*|..|*/..)
        die "archive contains unsafe path: ${entry}"
        ;;
    esac
  done < <(tar -tzf "${archive_path}")

  while IFS= read -r metadata; do
    entry_type="${metadata:0:1}"
    case "${entry_type}" in
      -|d)
        ;;
      *)
        die "archive contains unsupported entry type: ${metadata}"
        ;;
    esac
  done < <(tar -tvzf "${archive_path}")
}

reject_unsafe_tree_entries() {
  local source_dir="$1"
  local first_unsafe_entry

  first_unsafe_entry="$(find "${source_dir}" \( \( ! -type f ! -type d \) -o \( -type f -links +1 \) -o \( -perm /6000 \) -o \( -perm /0002 \) \) -print -quit)"
  if [[ -n "${first_unsafe_entry}" ]]; then
    die "tree contains non-regular, hardlinked, or unsafe-mode entry: ${first_unsafe_entry#"${source_dir}/"}"
  fi
}

read_previous_release() {
  local marker
  if [[ -f "${CURRENT_DIR}/${RELEASE_MARKER}" && ! -L "${CURRENT_DIR}/${RELEASE_MARKER}" ]]; then
    marker="$(<"${CURRENT_DIR}/${RELEASE_MARKER}")"
    if [[ "${marker}" =~ ^[A-Za-z0-9._-]{7,128}$ ]]; then
      printf '%s\n' "${marker}"
    fi
  fi
}

validate_source_tree() {
  local source_dir="$1"
  local required_file

  for required_file in Makefile docker-compose.yml config.yml; do
    if [[ ! -f "${source_dir}/${required_file}" ]]; then
      die "archive is missing required file: ${required_file}"
    fi
  done

  if [[ ! -d "${source_dir}/cmd/paradiced-server" ]]; then
    die "archive is missing required directory: cmd/paradiced-server"
  fi
}

preserve_runtime_dirs() {
  local runtime_dir
  if [[ "${runtime_preserved}" -eq 1 ]]; then
    return
  fi

  ensure_shared_runtime_root_safe
  mkdir -p "${SHARED_RUNTIME_DIR}"
  ensure_shared_runtime_root_safe

  for runtime_dir in "${RUNTIME_DIRS[@]}"; do
    ensure_runtime_dir_safe "${runtime_dir}"
    mkdir -p "${SHARED_RUNTIME_DIR}/${runtime_dir}"
    if [[ -d "${CURRENT_DIR}/${runtime_dir}" ]]; then
      reject_unsafe_tree_entries "${CURRENT_DIR}/${runtime_dir}"
      rsync "${RSYNC_COPY_OPTIONS[@]}" --delete "${CURRENT_DIR}/${runtime_dir}/" "${SHARED_RUNTIME_DIR}/${runtime_dir}/"
    fi
  done

  runtime_preserved=1
}

restore_runtime_dirs() {
  local runtime_dir
  ensure_shared_runtime_root_safe

  for runtime_dir in "${RUNTIME_DIRS[@]}"; do
    ensure_runtime_dir_safe "${runtime_dir}"
    mkdir -p "${SHARED_RUNTIME_DIR}/${runtime_dir}" "${CURRENT_DIR}/${runtime_dir}"
    ensure_runtime_dir_safe "${runtime_dir}"
    reject_unsafe_tree_entries "${SHARED_RUNTIME_DIR}/${runtime_dir}"
    rsync "${RSYNC_COPY_OPTIONS[@]}" --delete "${SHARED_RUNTIME_DIR}/${runtime_dir}/" "${CURRENT_DIR}/${runtime_dir}/"
  done
}

sync_release_snapshot() {
  local source_dir="$1"
  local release_dir="$2"

  ensure_directory_path_safe "${release_dir}"
  mkdir -p "${release_dir}"
  ensure_directory_path_safe "${release_dir}"
  rsync "${RSYNC_COPY_OPTIONS[@]}" --delete --delete-excluded "${RSYNC_EXCLUDES[@]}" "${source_dir}/" "${release_dir}/"
  ensure_directory_path_safe "${release_dir}"
}

sync_current_from_release() {
  local release="$1"
  local release_dir="${RELEASES_DIR}/${release}"

  if [[ ! -d "${release_dir}" ]]; then
    die "release snapshot does not exist: ${release_dir}"
  fi
  reject_unsafe_tree_entries "${release_dir}"

  ensure_current_real_directory
  mkdir -p "${CURRENT_DIR}"
  chown root:root "${CURRENT_DIR}"
  chmod 755 "${CURRENT_DIR}"
  ensure_current_real_directory
  preserve_runtime_dirs
  current_mutated=1
  rsync "${RSYNC_COPY_OPTIONS[@]}" --delete "${RSYNC_EXCLUDES[@]}" "${release_dir}/" "${CURRENT_DIR}/"
  restore_runtime_dirs
  printf '%s\n' "${release}" > "${CURRENT_DIR}/${RELEASE_MARKER}"
}

run_rebuild() {
  mkdir -p \
    "${CURRENT_DIR}/modules" \
    "${CURRENT_DIR}/.gomodcache" \
    "${CURRENT_DIR}/.gocache" \
    "${CURRENT_DIR}/.gopath"

  docker run --rm \
    --entrypoint /bin/sh \
    -e GOCACHE=/workspace/.gocache \
    -e GOMODCACHE=/workspace/.gomodcache \
    -e GOPATH=/workspace/.gopath \
    -e GOPROXY=https://goproxy.cn,direct \
    -e HOME=/tmp \
    -v "${CURRENT_DIR}:/workspace" \
    -w /workspace \
    heroiclabs/nakama-pluginbuilder:3.22.0 \
    -lc 'CGO_ENABLED=1 /usr/local/go/bin/go build -buildvcs=false --trimpath --buildmode=plugin -o ./modules/paradiced-server.so ./cmd/paradiced-server'

  (
    cd "${CURRENT_DIR}"
    COMPOSE_PROJECT_NAME=paradiced docker compose up -d --no-deps --force-recreate nakama cron-cleanup
  )
}

attempt_rollback() {
  if [[ -z "${previous_release}" ]]; then
    info "no previous release marker found; rollback skipped"
    return 1
  fi

  if [[ ! -d "${RELEASES_DIR}/${previous_release}" ]]; then
    info "previous release snapshot missing: ${RELEASES_DIR}/${previous_release}"
    return 1
  fi

  rollback_in_progress=1
  set +e
  (
    set -Eeuo pipefail
    trap - ERR
    sync_current_from_release "${previous_release}"
    run_rebuild
  )
  local rollback_status=$?
  set -e
  rollback_in_progress=0

  return "${rollback_status}"
}

on_error() {
  local status=$?
  local line_number="${1:-unknown}"

  if [[ "${rollback_in_progress}" -eq 1 ]]; then
    exit "${status}"
  fi

  if [[ "${current_mutated}" -eq 1 ]]; then
    info "deployment failed at line ${line_number}; attempting rollback"
    if attempt_rollback; then
      info "rollback completed: ${previous_release}"
    else
      info "rollback failed or unavailable"
    fi
  else
    info "deployment failed at line ${line_number}; current release was not changed"
  fi

  exit "${status}"
}
trap 'on_error "${LINENO}"' ERR

previous_release="$(read_previous_release || true)"
staging_dir="$(mktemp -d "${STAGING_ROOT}/${release_id}.XXXXXX")"

validate_archive_entries
info "extracting archive for release ${release_id}"
tar --no-same-owner --no-same-permissions -xzf "${archive_path}" -C "${staging_dir}"

source_dir="${staging_dir}"
if [[ ! -e "${source_dir}/Makefile" ]]; then
  shopt -s nullglob dotglob
  extracted_entries=("${staging_dir}"/*)
  shopt -u nullglob dotglob
  if [[ "${#extracted_entries[@]}" -eq 1 && -d "${extracted_entries[0]}" ]]; then
    source_dir="${extracted_entries[0]}"
  fi
fi

validate_source_tree "${source_dir}"
reject_unsafe_tree_entries "${source_dir}"
sync_release_snapshot "${source_dir}" "${RELEASES_DIR}/${release_id}"
sync_current_from_release "${release_id}"
run_rebuild

trap - ERR
info "deployment complete: ${release_id}"
