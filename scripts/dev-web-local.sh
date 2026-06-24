#!/usr/bin/env bash
# SPDX-License-Identifier: MIT
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCAL_DIR="${ROOT_DIR}/.local"

NAMESPACE="${DEV_LOCAL_NAMESPACE:-open-idb}"
PG_PORT="${DEV_LOCAL_PG_PORT:-15432}"
BACKEND_PORT="${DEV_LOCAL_BACKEND_PORT:-18080}"
WEB_PORT="${DEV_LOCAL_WEB_PORT:-5180}"
DATABASE_URL="${DEV_LOCAL_DATABASE_URL:-postgres://idbridge:idbridge-dev@127.0.0.1:${PG_PORT}/idbridge?sslmode=disable}"
GOCACHE_DIR="${GOCACHE:-${LOCAL_DIR}/go-build}"

PG_PID="${LOCAL_DIR}/dev-web-local-postgres.pid"
BACKEND_PID="${LOCAL_DIR}/dev-web-local-backend.pid"
WEB_PID="${LOCAL_DIR}/dev-web-local-web.pid"

PG_LOG="${LOCAL_DIR}/dev-web-local-postgres.log"
BACKEND_LOG="${LOCAL_DIR}/dev-web-local-backend.log"
WEB_LOG="${LOCAL_DIR}/dev-web-local-web.log"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[dev-web-local] missing dependency: $1" >&2
    exit 1
  }
}

pid_running() {
  local pidfile="$1"
  [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" >/dev/null 2>&1
}

stop_pid() {
  local pidfile="$1"
  local name="$2"
  local ownedfile="${pidfile}.owned"
  if pid_running "$pidfile" && [ -f "$ownedfile" ]; then
    local pid
    pid="$(cat "$pidfile")"
    echo "[dev-web-local] stopping ${name} pid=${pid}"
    kill -TERM "$pid" >/dev/null 2>&1 || true
    sleep 1
    kill -0 "$pid" >/dev/null 2>&1 && kill -KILL "$pid" >/dev/null 2>&1 || true
  elif pid_running "$pidfile"; then
    echo "[dev-web-local] leaving reused ${name} pid=$(cat "$pidfile") running"
  fi
  rm -f "$pidfile" "$ownedfile"
}

force_stop_port() {
  local port="$1"
  local name="$2"
  local pid
  pid="$(port_listen_pid "$port")"
  if [ -n "$pid" ]; then
    echo "[dev-web-local] stopping ${name} pid=${pid} port=${port}"
    kill -TERM "$pid" >/dev/null 2>&1 || true
    sleep 1
    kill -0 "$pid" >/dev/null 2>&1 && kill -KILL "$pid" >/dev/null 2>&1 || true
  fi
}

wait_tcp() {
  local host="$1"
  local port="$2"
  local name="$3"
  local deadline=$((SECONDS + 30))
  while [ "$SECONDS" -lt "$deadline" ]; do
    if nc -z "$host" "$port" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "[dev-web-local] ${name} did not open ${host}:${port}" >&2
  return 1
}

wait_http() {
  local url="$1"
  local name="$2"
  local deadline=$((SECONDS + 45))
  while [ "$SECONDS" -lt "$deadline" ]; do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "[dev-web-local] ${name} did not become ready: ${url}" >&2
  return 1
}

wait_stable_http() {
  local url="$1"
  local name="$2"
  local pidfile="$3"
  local logfile="$4"
  wait_http "$url" "$name"
  sleep 2
  assert_started "$name" "$pidfile" "$logfile"
  wait_http "$url" "$name"
}

port_listen_pid() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -tiTCP:"$port" -sTCP:LISTEN -n -P 2>/dev/null | head -n 1 || true
  fi
}

port_in_use() {
  local port="$1"
  [ -n "$(port_listen_pid "$port")" ]
}

fail_port_in_use() {
  local port="$1"
  local name="$2"
  echo "[dev-web-local] port ${port} is already in use, but it does not look like the expected ${name}" >&2
  lsof -iTCP:"$port" -sTCP:LISTEN -n -P >&2 || true
  exit 1
}

postgres_ready() {
  (
    cd "${ROOT_DIR}/backend"
    GOCACHE="$GOCACHE_DIR" go run github.com/pressly/goose/v3/cmd/goose@v3.22.1 \
      -dir migrations postgres "$DATABASE_URL" status >/dev/null 2>&1
  )
}

backend_ready() {
  curl -fsS --max-time 2 "http://localhost:${BACKEND_PORT}/healthz" >/dev/null 2>&1
}

web_ready() {
  curl -fsS --max-time 2 "http://localhost:${WEB_PORT}/" >/dev/null 2>&1
}

ensure_port_available_for_new_process() {
  local port="$1"
  local pidfile="$2"
  local name="$3"
  if pid_running "$pidfile"; then
    return
  fi
  if port_in_use "$port"; then
    fail_port_in_use "$port" "$name"
  fi
}

ensure_web_dependencies() {
  if [ ! -d "${ROOT_DIR}/web/node_modules" ]; then
    echo "[dev-web-local] installing web dependencies"
    (cd "${ROOT_DIR}/web" && npm install --legacy-peer-deps)
  fi
}

ensure_service_or_reuse() {
  local name="$1"
  local port="$2"
  local pidfile="$3"
  local readiness_fn="$4"

  if pid_running "$pidfile"; then
    if "$readiness_fn"; then
      echo "[dev-web-local] ${name} already running pid=$(cat "$pidfile") port=${port}"
      return 0
    fi
    echo "[dev-web-local] stale ${name} pid file found; removing ${pidfile}"
    rm -f "$pidfile"
  fi

  if port_in_use "$port"; then
    if "$readiness_fn"; then
      local reused_pid
      reused_pid="$(port_listen_pid "$port")"
      echo "[dev-web-local] reusing existing ${name} pid=${reused_pid} port=${port}"
      echo "$reused_pid" >"$pidfile"
      rm -f "${pidfile}.owned"
      return 0
    fi
    fail_port_in_use "$port" "$name"
  fi

  return 1
}

warn_legacy_react_frontend() {
  local react_pid
  react_pid="$(port_listen_pid 5173)"
  if [ -n "$react_pid" ] && [ "$WEB_PORT" != "5173" ]; then
    echo "[dev-web-local] note: port 5173 is already in use; this script starts Svelte web on ${WEB_PORT} and does not manage the old frontend"
  fi
}

print_tail_hint() {
  local name="$1"
  local logfile="$2"
  if [ -f "$logfile" ]; then
    echo "[dev-web-local] ${name} log tail:"
    tail -n 20 "$logfile" || true
  fi
}

assert_started() {
  local name="$1"
  local pidfile="$2"
  local logfile="$3"
  if ! pid_running "$pidfile"; then
    echo "[dev-web-local] ${name} process exited during startup" >&2
    print_tail_hint "$name" "$logfile" >&2
    exit 1
  fi
}

start_postgres_forward() {
  if ensure_service_or_reuse "postgres" "$PG_PORT" "$PG_PID" postgres_ready; then
    return
  fi
  echo "[dev-web-local] starting postgres port-forward :${PG_PORT}"
  nohup kubectl -n "$NAMESPACE" port-forward svc/postgres "${PG_PORT}:5432" >"$PG_LOG" 2>&1 &
  echo $! >"$PG_PID"
  touch "${PG_PID}.owned"
  wait_tcp 127.0.0.1 "$PG_PORT" "postgres port-forward"
  wait_tcp 127.0.0.1 "$PG_PORT" "postgres"
}

run_migrations() {
  echo "[dev-web-local] applying backend migrations"
  (
    cd "${ROOT_DIR}/backend"
    GOCACHE="$GOCACHE_DIR" go run github.com/pressly/goose/v3/cmd/goose@v3.22.1 \
      -dir migrations postgres "$DATABASE_URL" up
  )
}

start_backend() {
  if ensure_service_or_reuse "backend" "$BACKEND_PORT" "$BACKEND_PID" backend_ready; then
    return
  fi
  ensure_port_available_for_new_process "$BACKEND_PORT" "$BACKEND_PID" "backend"
  echo "[dev-web-local] starting backend :${BACKEND_PORT}"
  nohup bash -lc "
    cd "${ROOT_DIR}/backend"
    export GOCACHE='${GOCACHE_DIR}'
    export DATABASE_URL='${DATABASE_URL}'
    export IDB_HTTP_ADDR=':${BACKEND_PORT}'
    export IDB_OIDC_ISSUER='http://localhost:${BACKEND_PORT}'
    export IDB_WEB_BASE_URL='http://localhost:${WEB_PORT}'
    export IDB_FEISHU_REDIRECT_URI='http://localhost:${WEB_PORT}/api/auth/feishu/callback'
    go run ./cmd/idbridge
  " >"$BACKEND_LOG" 2>&1 &
  echo $! >"$BACKEND_PID"
  touch "${BACKEND_PID}.owned"
  sleep 1
  assert_started "backend" "$BACKEND_PID" "$BACKEND_LOG"
  wait_stable_http "http://localhost:${BACKEND_PORT}/healthz" "backend" "$BACKEND_PID" "$BACKEND_LOG"
}

start_web() {
  if ensure_service_or_reuse "web" "$WEB_PORT" "$WEB_PID" web_ready; then
    return
  fi
  ensure_port_available_for_new_process "$WEB_PORT" "$WEB_PID" "web"
  ensure_web_dependencies
  echo
  echo "[dev-web-local] ready"
  echo "  web:     http://localhost:${WEB_PORT}/"
  echo "  backend: http://localhost:${BACKEND_PORT}/healthz"
  echo "  logs:    ${LOCAL_DIR}/dev-web-local-*.log"
  echo
  echo "[dev-web-local] starting Svelte web in foreground; press Ctrl+C to stop"
  (
    cd "${ROOT_DIR}/web"
    PUBLIC_API_TARGET="http://localhost:${BACKEND_PORT}" \
    VITE_API_TARGET="http://localhost:${BACKEND_PORT}" \
    npm run dev -- --host 0.0.0.0 --port "$WEB_PORT"
  )
}

start_all() {
  mkdir -p "$LOCAL_DIR" "$GOCACHE_DIR"
  require_cmd kubectl
  require_cmd go
  require_cmd npm
  require_cmd curl
  require_cmd nc

  kubectl version --request-timeout=5s >/dev/null
  kubectl -n "$NAMESPACE" get svc postgres >/dev/null

  warn_legacy_react_frontend
  start_postgres_forward
  run_migrations
  start_backend
  start_web
}

stop_all() {
  stop_pid "$WEB_PID" "web"
  stop_pid "$BACKEND_PID" "backend"
  stop_pid "$PG_PID" "postgres port-forward"
}

restart_all() {
  stop_all
  force_stop_port "$WEB_PORT" "web"
  force_stop_port "$BACKEND_PORT" "backend"
  force_stop_port "$PG_PORT" "postgres port-forward"
  start_all
}

reset_database() {
  mkdir -p "$LOCAL_DIR" "$GOCACHE_DIR"
  require_cmd kubectl
  require_cmd go
  require_cmd curl
  require_cmd nc

  echo "[dev-web-local] resetting postgres deployment; local dev data will be deleted"
  stop_pid "$WEB_PID" "web"
  stop_pid "$BACKEND_PID" "backend"
  stop_pid "$PG_PID" "postgres port-forward"
  force_stop_port "$WEB_PORT" "web"
  force_stop_port "$BACKEND_PORT" "backend"
  force_stop_port "$PG_PORT" "postgres port-forward"
  kubectl -n "$NAMESPACE" rollout restart deployment/postgres
  kubectl -n "$NAMESPACE" rollout status deployment/postgres --timeout=120s
  start_postgres_forward
  run_migrations
  start_backend
  echo "[dev-web-local] database reset complete"
}

status_all() {
  status_one "postgres" "$PG_PID" "$PG_PORT" postgres_ready
  status_one "backend" "$BACKEND_PID" "$BACKEND_PORT" backend_ready
  status_one "web" "$WEB_PID" "$WEB_PORT" web_ready
}

status_one() {
  local name="$1"
  local pidfile="$2"
  local port="$3"
  local readiness_fn="$4"
  local pid=""
  local source="pidfile"

  if pid_running "$pidfile"; then
    pid="$(cat "$pidfile")"
    if [ -f "${pidfile}.owned" ]; then
      source="pidfile"
    else
      source="reused-pidfile"
    fi
  else
    pid="$(port_listen_pid "$port")"
    source="port"
  fi

  if [ -n "$pid" ]; then
    if "$readiness_fn"; then
      echo "${name}: running pid=${pid} port=${port} source=${source} ready=yes"
    else
      echo "${name}: running pid=${pid} port=${port} source=${source} ready=no"
    fi
  else
    echo "${name}: stopped port=${port}"
  fi
}

logs_all() {
  tail -n 80 "$PG_LOG" "$BACKEND_LOG" "$WEB_LOG" 2>/dev/null || true
}

case "${1:-start}" in
  start)
    start_all
    ;;
  stop)
    stop_all
    ;;
  restart)
    restart_all
    ;;
  reset-db)
    reset_database
    ;;
  status)
    status_all
    ;;
  logs)
    logs_all
    ;;
  *)
    echo "Usage: $0 [start|stop|restart|reset-db|status|logs]" >&2
    exit 2
    ;;
esac
