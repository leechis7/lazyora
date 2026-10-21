#!/bin/sh
set -eu
cd "$(dirname "$0")"

# Always release the test DB memory when this launcher exits. Keep its data volume.
cleanup() {
    run_status=$?
    trap - 0 HUP INT TERM
    printf '\n%s\n' 'Oracle 시험 환경을 종료합니다(DB 데이터는 보존합니다).'
    if ! docker compose -f compose.yaml down; then
        printf '%s\n' 'Oracle 종료에 실패했습니다. make -C test/oracle down으로 다시 시도하세요.' >&2
        if [ "$run_status" -eq 0 ]; then
            run_status=1
        fi
    fi
    exit "$run_status"
}
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

# Create local settings if needed, then start (or reuse) the DB and wait until
# its healthcheck succeeds before opening the app. Never pass passwords as args.
sh env.sh
. ./.env
: "${APP_USER_PASSWORD:?로컬 .env의 APP_USER_PASSWORD를 설정하세요}"
trap cleanup 0
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
printf '%s\n' 'Oracle 시험 환경을 준비합니다(DB 접속 준비까지 기다립니다).'
docker compose -f compose.yaml up -d --wait --wait-timeout 600 oracle

export LAZYORA_HOST=127.0.0.1
export LAZYORA_PORT="${ORACLE_PORT:-1521}"
export LAZYORA_USER=lazyora
export LAZYORA_PASSWORD="${APP_USER_PASSWORD:?로컬 .env의 APP_USER_PASSWORD를 설정하세요}"
export LAZYORA_NAME=FREEPDB1
"$binary" "$@"
