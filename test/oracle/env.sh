#!/bin/sh
set -eu

# Always write beside compose.yaml, regardless of the caller's working directory.
cd "$(dirname "$0")"
if [ -e .env ]; then
    printf '%s\n' '.env가 이미 있어 기존 설정을 유지합니다.'
    exit 0
fi

umask 077
admin_password=$(od -An -N18 -tx1 /dev/urandom | tr -d ' \n')
app_password=$(od -An -N18 -tx1 /dev/urandom | tr -d ' \n')
(
    set -C
    printf 'ORACLE_PASSWORD=%s\nAPP_USER_PASSWORD=%s\nORACLE_PORT=1521\n' \
        "$admin_password" "$app_password" > .env
)
printf '%s\n' '로컬 테스트 비밀번호를 .env에 생성했습니다. Git에는 포함되지 않습니다.'
