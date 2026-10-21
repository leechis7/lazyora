#!/bin/sh
set -eu
cd "$(dirname "$0")"
if [ ! -f .env ]; then
    printf '%s\n' '먼저 make -C test/oracle up으로 시험 DB를 준비하세요.' >&2
    exit 1
fi
set -a
. ./.env
set +a
cd ../..
python3 test/oracle/tui_smoke.py
python3 test/oracle/tui_cells_smoke.py
