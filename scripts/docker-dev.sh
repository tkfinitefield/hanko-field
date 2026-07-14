#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

export PATH="${HOST_PATH:+${HOST_PATH}:}${PATH}"
export CARGO_BUILD_JOBS="${CARGO_BUILD_JOBS:-1}"
export CARGO_TARGET_DIR="${CARGO_TARGET_DIR:-/tmp/hanko-field-cargo-target}"

api_pid=""
admin_pid=""
web_pid=""

cleanup() {
  local status=$?
  trap - EXIT INT TERM

  for pid in "${api_pid}" "${admin_pid}" "${web_pid}"; do
    if [[ -n "${pid}" ]]; then
      kill "${pid}" 2>/dev/null || true
    fi
  done

  for pid in "${api_pid}" "${admin_pid}" "${web_pid}"; do
    if [[ -n "${pid}" ]]; then
      wait "${pid}" 2>/dev/null || true
    fi
  done

  exit "${status}"
}

trap cleanup EXIT INT TERM

./scripts/ironframe.sh build \
  -i admin/static/input.css \
  -o admin/static/style.css \
  "admin/templates/**/*.html" \
  "admin/static/*.js"
./scripts/ironframe.sh build \
  -i web/static/input.css \
  -o web/static/style.css \
  "web/templates/**/*.html" \
  "web/static/*.js"

cargo build --manifest-path api/Cargo.toml --bin hanko-field-api
cargo build --manifest-path admin/Cargo.toml
cargo build --manifest-path web/Cargo.toml

(
  API_SERVER_PORT="${API_SERVER_PORT:-3050}" exec "${CARGO_TARGET_DIR}/debug/hanko-field-api"
) &
api_pid=$!

(
  ADMIN_HTTP_ADDR=":${ADMIN_PORT:-3051}" \
    HANKO_ADMIN_MODE="${HANKO_ADMIN_MODE:-mock}" \
    HANKO_ADMIN_LOCALE="${HANKO_ADMIN_LOCALE:-ja}" \
    exec "${CARGO_TARGET_DIR}/debug/hanko-field-admin"
) &
admin_pid=$!

(
  HANKO_WEB_PORT="${HANKO_WEB_PORT:-3052}" \
    HANKO_WEB_MODE="${HANKO_WEB_MODE:-mock}" \
    HANKO_WEB_LOCALE="${HANKO_WEB_LOCALE:-ja}" \
    exec "${CARGO_TARGET_DIR}/debug/hanko-field-web"
) &
web_pid=$!

set +e
wait -n
status=$?
set -e

exit "${status}"
