#!/bin/sh
set -eu
cd "$(dirname "$0")"
case "${1:---local}" in
    --local)
        if [ ! -f .env ]; then
            printf '%s\n' '먼저 make -C test/oracle up으로 시험 DB를 준비하세요.' >&2
            exit 1
        fi
        set -a
        . ./.env
        set +a
        export LAZYORA_COMPAT_HOST=127.0.0.1 LAZYORA_COMPAT_PORT="$ORACLE_PORT"
        export LAZYORA_COMPAT_USER=lazyora LAZYORA_COMPAT_PASSWORD="$APP_USER_PASSWORD"
        export LAZYORA_COMPAT_MODE=service LAZYORA_COMPAT_NAME=FREEPDB1
        ;;
    --external) ;; # Explicit opt-in; never load or override external credentials.
    *) printf '%s\n' '사용법: sh test/oracle/compat-test.sh [--local|--external]' >&2; exit 2 ;;
esac
export LAZYORA_COMPAT=1
cd ../..
go test ./test/oracle -run '^TestOracleCompatibilityIntegration$' -v -count=1 -timeout=3m
