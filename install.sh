#!/bin/sh

set -eu

DEFAULT_GITHUB_REPO="wtgoku-create/popiartcli"
DEFAULT_GITEE_REPO="wattx/popiartcli"
SOURCE="${POPIART_SOURCE:-github}"
REPO="${POPIART_REPO:-}"
REPO_INFERRED_TAG=""
BINARY="popiart"
CLI_ONLY=0
RUN_BOOTSTRAP=0
WITH_DEFAULT_SKILLS=0
NO_AGENT_CONFIG=0
KEY=""
ENDPOINT=""
PROJECT=""
AGENTS=""
COMPLETIONS=""
VERSION_FLAG=""

log() {
  printf '%s\n' "$*" >&2
}

fail() {
  log "error: $*"
  exit 1
}

trim_github_archive_tag() {
  tag_name="$1"
  tag_name="${tag_name%.tar.gz}"
  tag_name="${tag_name%.tgz}"
  tag_name="${tag_name%.zip}"
  printf '%s\n' "${tag_name}"
}

normalize_source() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    ""|github) printf 'github\n' ;;
    gitee) printf 'gitee\n' ;;
    *) fail "unsupported source: $1 (expected github or gitee)" ;;
  esac
}

default_repo_for_source() {
  case "$1" in
    gitee) printf '%s\n' "${DEFAULT_GITEE_REPO}" ;;
    *) printf '%s\n' "${DEFAULT_GITHUB_REPO}" ;;
  esac
}

normalize_repo_input() {
  input="$1"
  input="${input%/}"
  REPO_INFERRED_TAG=""

  if [ -z "${input}" ]; then
    printf '%s\n' "$(default_repo_for_source "${SOURCE}")"
    return
  fi

  case "${input}" in
    https://github.com/*|http://github.com/*|github.com/*)
      SOURCE="github"
      normalized="${input#https://}"
      normalized="${normalized#http://}"
      normalized="${normalized#github.com/}"
      normalized="${normalized%%\?*}"
      normalized="${normalized%%#*}"
      owner="${normalized%%/*}"
      rest="${normalized#*/}"
      [ -n "${owner}" ] && [ "${rest}" != "${normalized}" ] || fail "invalid GitHub repository URL: ${input}"
      name="${rest%%/*}"
      remainder=""
      if [ "${rest}" != "${name}" ]; then
        remainder="${rest#*/}"
      fi
      name="${name%.git}"
      [ -n "${name}" ] || fail "invalid GitHub repository URL: ${input}"
      repo="${owner}/${name}"

      case "${remainder}" in
        releases/tag/*)
          REPO_INFERRED_TAG="${remainder#releases/tag/}"
          ;;
        archive/refs/tags/*)
          REPO_INFERRED_TAG="$(trim_github_archive_tag "${remainder#archive/refs/tags/}")"
          ;;
        archive/*)
          REPO_INFERRED_TAG="$(trim_github_archive_tag "${remainder#archive/}")"
          ;;
      esac

      printf '%s\n' "${repo}"
      ;;
    https://gitee.com/*|http://gitee.com/*|gitee.com/*)
      SOURCE="gitee"
      normalized="${input#https://}"
      normalized="${normalized#http://}"
      normalized="${normalized#gitee.com/}"
      normalized="${normalized%%\?*}"
      normalized="${normalized%%#*}"
      owner="${normalized%%/*}"
      rest="${normalized#*/}"
      [ -n "${owner}" ] && [ "${rest}" != "${normalized}" ] || fail "invalid Gitee repository URL: ${input}"
      name="${rest%%/*}"
      remainder=""
      if [ "${rest}" != "${name}" ]; then
        remainder="${rest#*/}"
      fi
      name="${name%.git}"
      [ -n "${name}" ] || fail "invalid Gitee repository URL: ${input}"
      repo="${owner}/${name}"

      case "${remainder}" in
        releases/tag/*)
          REPO_INFERRED_TAG="${remainder#releases/tag/}"
          ;;
        archive/refs/tags/*)
          REPO_INFERRED_TAG="$(trim_github_archive_tag "${remainder#archive/refs/tags/}")"
          ;;
        archive/*)
          REPO_INFERRED_TAG="$(trim_github_archive_tag "${remainder#archive/}")"
          ;;
      esac

      printf '%s\n' "${repo}"
      ;;
    *)
      repo="${input%.git}"
      owner="${repo%%/*}"
      rest="${repo#*/}"
      [ -n "${owner}" ] && [ "${rest}" != "${repo}" ] || fail "expected repository in owner/name format"
      case "${rest}" in
        */*) fail "expected repository in owner/name format" ;;
      esac
      printf '%s\n' "${owner}/${rest}"
      ;;
  esac
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

append_line() {
  value="$1"
  current="$2"
  if [ -z "${current}" ]; then
    printf '%s' "${value}"
  else
    printf '%s\n%s' "${current}" "${value}"
  fi
}

usage() {
  cat <<'EOF'
Usage: install.sh [options]

Install popiart from GitHub or Gitee Releases.
By default the script installs the CLI only.

Options:
  --source <name>        Download source: github or gitee
  --cli-only             Compatibility alias: install the CLI only
  --bootstrap            Run `popiart bootstrap` after installation
  --with-default-skills  Generate the default remote skill discovery profile
  --agent <name>         Generate bootstrap files for an agent (repeatable)
  --completion <shell>   Generate shell completion for bash/zsh/fish/powershell
  --no-agent-config      Skip agent env file generation
  --key <key>            Save an API key during bootstrap
  --endpoint <url>       Persist endpoint during bootstrap
  --project <id>         Persist project during bootstrap
  --version <tag>        Install a specific release, for example v0.1.0
  --help                 Show this help message
EOF
}

resolve_os() {
  case "$(uname -s)" in
    Darwin) printf 'darwin\n' ;;
    Linux) printf 'linux\n' ;;
    *) fail "unsupported operating system: $(uname -s)" ;;
  esac
}

resolve_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_latest_tag() {
  need_cmd curl
  case "${SOURCE}" in
    gitee)
      latest_json="$(curl -fsSL "https://gitee.com/api/v5/repos/${REPO}/releases/latest")" || fail "failed to resolve latest release tag from Gitee"
      latest_tag="$(printf '%s' "${latest_json}" | tr -d '\r\n' | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
      ;;
    *)
      latest_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")"
      latest_tag="${latest_url##*/}"
      latest_tag="${latest_tag%%\?*}"
      ;;
  esac
  [ -n "${latest_tag}" ] || fail "failed to resolve latest release tag"
  printf '%s\n' "${latest_tag}"
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi

  fail "missing checksum tool: sha256sum or shasum"
}

install_dir() {
  if [ -n "${BINDIR:-}" ]; then
    printf '%s\n' "${BINDIR}"
    return
  fi

  if [ "$(resolve_os)" = "darwin" ]; then
    if command -v brew >/dev/null 2>&1; then
      brew_bindir="$(brew --prefix 2>/dev/null)/bin"
      if [ -n "${brew_bindir}" ] && [ -d "${brew_bindir}" ] && [ -w "${brew_bindir}" ]; then
        printf '%s\n' "${brew_bindir}"
        return
      fi
    fi

    if [ -d /opt/homebrew/bin ] && [ -w /opt/homebrew/bin ]; then
      printf '/opt/homebrew/bin\n'
      return
    fi
  fi

  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    printf '/usr/local/bin\n'
    return
  fi

  if [ -n "${HOME:-}" ]; then
    printf '%s\n' "${HOME}/.local/bin"
    return
  fi

  if [ "$(resolve_os)" = "darwin" ] && [ -d /opt/homebrew/bin ]; then
    printf '/opt/homebrew/bin\n'
    return
  fi

  printf '/usr/local/bin\n'
}

profile_file_hint() {
  shell_name="${SHELL:-}"
  shell_name="${shell_name##*/}"
  case "${shell_name}" in
    zsh)
      printf '%s\n' "${HOME}/.zprofile"
      ;;
    bash)
      printf '%s\n' "${HOME}/.bash_profile"
      ;;
    fish)
      printf '%s\n' "${HOME}/.config/fish/config.fish"
      ;;
    *)
      printf '%s\n' "${HOME}/.profile"
      ;;
  esac
}

path_export_snippet() {
  shell_name="${SHELL:-}"
  shell_name="${shell_name##*/}"
  case "${shell_name}" in
    fish)
      printf 'fish_add_path "%s"\n' "${bindir}"
      ;;
    *)
      printf 'export PATH="%s:$PATH"\n' "${bindir}"
      ;;
  esac
}

detect_completion_shell() {
  shell_name="${SHELL:-}"
  shell_name="${shell_name##*/}"
  case "${shell_name}" in
    bash|zsh|fish) printf '%s\n' "${shell_name}" ;;
    pwsh|powershell) printf 'powershell\n' ;;
    *) return 1 ;;
  esac
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --cli-only)
      CLI_ONLY=1
      shift
      ;;
    --source)
      [ "$#" -ge 2 ] || fail "missing value for --source"
      SOURCE="$2"
      shift 2
      ;;
    --bootstrap)
      RUN_BOOTSTRAP=1
      shift
      ;;
    --with-default-skills)
      WITH_DEFAULT_SKILLS=1
      shift
      ;;
    --no-agent-config)
      NO_AGENT_CONFIG=1
      shift
      ;;
    --agent)
      [ "$#" -ge 2 ] || fail "missing value for --agent"
      AGENTS="$(append_line "$2" "${AGENTS}")"
      shift 2
      ;;
    --completion)
      [ "$#" -ge 2 ] || fail "missing value for --completion"
      COMPLETIONS="$(append_line "$2" "${COMPLETIONS}")"
      shift 2
      ;;
    --key)
      [ "$#" -ge 2 ] || fail "missing value for --key"
      KEY="$2"
      shift 2
      ;;
    --endpoint)
      [ "$#" -ge 2 ] || fail "missing value for --endpoint"
      ENDPOINT="$2"
      shift 2
      ;;
    --project)
      [ "$#" -ge 2 ] || fail "missing value for --project"
      PROJECT="$2"
      shift 2
      ;;
    --version)
      [ "$#" -ge 2 ] || fail "missing value for --version"
      VERSION_FLAG="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "unknown option: $1"
      ;;
  esac
done

need_cmd curl
need_cmd tar
need_cmd install

SOURCE="$(normalize_source "${SOURCE}")"
REPO="$(normalize_repo_input "${REPO}")"

os="$(resolve_os)"
arch="$(resolve_arch)"
bindir="$(install_dir)"
tag="${VERSION:-}"

if [ -z "${tag}" ]; then
  tag="${VERSION_FLAG}"
fi

if [ -z "${tag}" ] && [ -n "${REPO_INFERRED_TAG}" ]; then
  tag="${REPO_INFERRED_TAG}"
fi

if [ -z "${tag}" ]; then
  tag="$(resolve_latest_tag)"
fi

version="${tag#v}"
archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
checksums="checksums.txt"
case "${SOURCE}" in
  gitee)
    base_url="https://gitee.com/${REPO}/releases/download/v${version}"
    ;;
  *)
    base_url="https://github.com/${REPO}/releases/download/v${version}"
    ;;
esac

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT INT TERM

archive_path="${tmpdir}/${archive}"
checksums_path="${tmpdir}/${checksums}"

log "downloading ${archive}"
curl -fsSL "${base_url}/${archive}" -o "${archive_path}" || fail "failed to download archive from ${base_url}/${archive}"
curl -fsSL "${base_url}/${checksums}" -o "${checksums_path}" || fail "failed to download checksums from ${base_url}/${checksums}"

expected_sha="$(awk -v file="${archive}" '$2 == file { print $1 }' "${checksums_path}")"
[ -n "${expected_sha}" ] || fail "checksum entry for ${archive} not found"

actual_sha="$(sha256_file "${archive_path}")"
[ "${expected_sha}" = "${actual_sha}" ] || fail "checksum mismatch for ${archive}"

tar -xzf "${archive_path}" -C "${tmpdir}" "${BINARY}" || fail "failed to extract ${archive}"

SUDO=""
if [ -d "${bindir}" ]; then
  if [ ! -w "${bindir}" ]; then
    command -v sudo >/dev/null 2>&1 || fail "no permission to write to ${bindir}; rerun with BINDIR set to a writable directory"
    SUDO="sudo"
  fi
else
  if ! mkdir -p "${bindir}" 2>/dev/null; then
    command -v sudo >/dev/null 2>&1 || fail "failed to create ${bindir}; rerun with BINDIR set to a writable directory"
    SUDO="sudo"
    ${SUDO} mkdir -p "${bindir}"
  fi
fi

${SUDO} install -m 0755 "${tmpdir}/${BINARY}" "${bindir}/${BINARY}"

log "installed ${BINARY} ${version} to ${bindir}/${BINARY}"

case ":${PATH}:" in
  *:"${bindir}":*) ;;
  *)
    log "warning: ${bindir} is not on PATH in the current shell"
    if [ -n "${HOME:-}" ]; then
      log "add this to $(profile_file_hint):"
    else
      log "add this to your shell profile:"
    fi
    log "  $(path_export_snippet)"
    log "then open a new terminal or run:"
    log "  $(path_export_snippet)"
    ;;
esac

if [ "${CLI_ONLY}" -eq 1 ] || [ "${RUN_BOOTSTRAP}" -eq 0 ]; then
  exit 0
fi

if [ -z "${COMPLETIONS}" ]; then
  if detected_shell="$(detect_completion_shell)"; then
    COMPLETIONS="${detected_shell}"
  fi
fi

set -- "${bindir}/${BINARY}"
[ -n "${ENDPOINT}" ] && set -- "$@" --endpoint "${ENDPOINT}"
[ -n "${PROJECT}" ] && set -- "$@" --project "${PROJECT}"
set -- "$@" --plain bootstrap
[ -n "${KEY}" ] && set -- "$@" --key "${KEY}"
[ "${WITH_DEFAULT_SKILLS}" -eq 1 ] && set -- "$@" --with-default-skills
[ "${NO_AGENT_CONFIG}" -eq 1 ] && set -- "$@" --no-agent-config

old_ifs="${IFS}"
IFS='
'
for agent in ${AGENTS}; do
  [ -n "${agent}" ] && set -- "$@" --agent "${agent}"
done
for shell_name in ${COMPLETIONS}; do
  [ -n "${shell_name}" ] && set -- "$@" --completion "${shell_name}"
done
IFS="${old_ifs}"

log "running bootstrap"
"$@"
