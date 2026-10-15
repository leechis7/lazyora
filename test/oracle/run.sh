#!/bin/sh
set -eu
cd "$(dirname "$0")"

if [ ! -f .env ]; then
    printf '%s\n' '먼저 프로젝트 루트에서 make -C test/oracle up을 실행하세요.' >&2
    exit 1
fi

# Load only the local test settings; do not put passwords in command arguments.
. ./.env

case "$(uname -s)" in
    Linux) target_os=linux ;;
    Darwin) target_os=darwin ;;
    *) printf '%s\n' '이 실행 스크립트는 Linux와 macOS에서 사용할 수 있습니다.' >&2; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64) target_arch=amd64 ;;
    aarch64|arm64) target_arch=arm64 ;;
    *) printf '%s\n' '지원하지 않는 CPU 아키텍처입니다.' >&2; exit 1 ;;
esac

binary="../../bin/lazyora-${target_os}-${target_arch}"
if [ ! -x "$binary" ]; then
    printf '%s\n' '먼저 프로젝트 루트에서 make build-native를 실행하세요.' >&2
    exit 1
fi

export LAZYORA_HOST=127.0.0.1
export LAZYORA_PORT="${ORACLE_PORT:-1521}"
export LAZYORA_USER=lazyora
export LAZYORA_PASSWORD="${APP_USER_PASSWORD:?로컬 .env의 APP_USER_PASSWORD를 설정하세요}"
export LAZYORA_NAME=FREEPDB1
exec "$binary" "$@"
