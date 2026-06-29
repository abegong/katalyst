#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
public_dir="${repo_root}/docs/public"
print_page="${public_dir}/print/index.html"
output_pdf="${DOCS_PDF_OUTPUT:-${public_dir}/katalyst-docs.pdf}"

if [[ ! -f "${print_page}" ]]; then
  echo "missing ${print_page}; run make docs-build or make docs-pdf first" >&2
  exit 1
fi

find_browser() {
  local candidates=(
    "${CHROME_BIN:-}"
    "google-chrome"
    "google-chrome-stable"
    "chromium"
    "chromium-browser"
    "microsoft-edge"
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
    "/Applications/Chromium.app/Contents/MacOS/Chromium"
    "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
  )

  local candidate
  for candidate in "${candidates[@]}"; do
    [[ -n "${candidate}" ]] || continue
    if command -v "${candidate}" >/dev/null 2>&1; then
      command -v "${candidate}"
      return 0
    fi
    if [[ -x "${candidate}" ]]; then
      printf '%s\n' "${candidate}"
      return 0
    fi
  done
}

browser="$(find_browser || true)"
if [[ -z "${browser}" ]]; then
  echo "could not find Chrome, Chromium, or Edge; set CHROME_BIN=/path/to/browser" >&2
  exit 1
fi

file_size() {
  stat -f%z "$1" 2>/dev/null || stat -c%s "$1"
}

port="$(python3 - <<'PY'
import socket

with socket.socket() as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
)"

python3 -m http.server "${port}" --bind 127.0.0.1 --directory "${public_dir}" >/tmp/katalyst-docs-pdf-http.log 2>&1 &
server_pid="$!"
profile_dir="$(mktemp -d)"
browser_log="$(mktemp)"
cleanup() {
  kill "${server_pid}" >/dev/null 2>&1 || true
  wait "${server_pid}" 2>/dev/null || true
  rm -rf "${profile_dir}"
  rm -f "${browser_log}"
}
trap cleanup EXIT

for _ in {1..40}; do
  if python3 - "$port" <<'PY' >/dev/null 2>&1
import http.client
import sys

conn = http.client.HTTPConnection("127.0.0.1", int(sys.argv[1]), timeout=0.2)
conn.request("GET", "/print/")
response = conn.getresponse()
sys.exit(0 if response.status < 500 else 1)
PY
  then
    break
  fi
  sleep 0.1
done

mkdir -p "$(dirname "${output_pdf}")"
rm -f "${output_pdf}"

"${browser}" \
  --headless=new \
  --disable-background-networking \
  --disable-component-update \
  --disable-default-apps \
  --disable-gpu \
  --disable-sync \
  --metrics-recording-only \
  --mute-audio \
  --no-sandbox \
  --user-data-dir="${profile_dir}" \
  --print-to-pdf="${output_pdf}" \
  --no-pdf-header-footer \
  --print-to-pdf-no-header \
  "http://127.0.0.1:${port}/print/" \
  >"${browser_log}" 2>&1 &
browser_pid="$!"

last_size=0
stable_count=0
for _ in {1..240}; do
  if [[ -f "${output_pdf}" ]]; then
    size="$(file_size "${output_pdf}")"
    if [[ "${size}" -gt 0 && "${size}" == "${last_size}" ]]; then
      stable_count=$((stable_count + 1))
    else
      stable_count=0
      last_size="${size}"
    fi

    if [[ "${stable_count}" -ge 4 ]]; then
      break
    fi
  fi

  if ! kill -0 "${browser_pid}" >/dev/null 2>&1; then
    wait "${browser_pid}" || {
      cat "${browser_log}" >&2
      exit 1
    }
    break
  fi

  sleep 0.25
done

if [[ ! -s "${output_pdf}" ]]; then
  kill "${browser_pid}" >/dev/null 2>&1 || true
  wait "${browser_pid}" 2>/dev/null || true
  cat "${browser_log}" >&2
  echo "browser did not write ${output_pdf}" >&2
  exit 1
fi

kill "${browser_pid}" >/dev/null 2>&1 || true
wait "${browser_pid}" 2>/dev/null || true

echo "wrote ${output_pdf}"
