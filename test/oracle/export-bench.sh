#!/bin/sh
set -eu
cd "$(dirname "$0")"
if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi
export LAZYORA_INTEGRATION=1 LAZYORA_EXPORT_BENCH=1
cd ../..
# Keep outputs on the workspace filesystem, not /tmp (often RAM-backed).
bench_dir=$(mktemp -d "${BENCH_DIR:-$PWD/test/oracle}/.lazyora-export-bench.XXXXXX")
trap 'rm -rf "$bench_dir"' EXIT HUP INT TERM
go test -c -o "$bench_dir/export.test" ./test/oracle
for source in ${BENCH_SOURCES:-generated oracle}; do
    for rows in ${BENCH_ROWS:-100000 1000000}; do
        for format in ${BENCH_FORMATS:-csv tsv sql xlsx}; do
            printf '측정: %s / %s / %s행\n' "$source" "$format" "$rows"
            LAZYORA_BENCH_SOURCE="$source" LAZYORA_BENCH_ROWS="$rows" LAZYORA_BENCH_FORMAT="$format" \
                TMPDIR="$bench_dir" "$bench_dir/export.test" -test.run '^TestOracleExportPerformance$' -test.v -test.timeout 15m
        done
    done
done
