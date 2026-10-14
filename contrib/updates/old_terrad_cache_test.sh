#!/bin/bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "$script_dir/old_terrad_cache.sh"
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT

binary="$test_dir/terrad"
stamp="$test_dir/terrad.version"
printf 'test binary v4.0.1\n' > "$binary"
revision=8d9d136c61b77601ce257a97d26e116812c0e54a
binary_sha=$(sha256sum "$binary" | awk '{print $1}')
printf 'version=v4.0.1\nrevision=%s\nsha256=%s\n' "$revision" "$binary_sha" > "$stamp"

old_terrad_cache_valid "$binary" "$stamp" v4.0.1 "$revision"
if old_terrad_cache_valid "$binary" "$stamp" v3.1.6 "$revision"; then
    echo 'cache unexpectedly accepted for a different OLD_VERSION' >&2
    exit 1
fi

if old_terrad_cache_valid "$binary" "$stamp" v4.0.1 0000000000000000000000000000000000000000; then
    echo 'cache accepted different immutable revision' >&2
    exit 1
fi
if old_terrad_cache_valid "$binary" "$stamp" v4.0.1 main; then
    echo 'cache accepted mutable revision' >&2
    exit 1
fi
printf 'modified binary\n' > "$binary"
if old_terrad_cache_valid "$binary" "$stamp" v4.0.1 "$revision"; then
    echo 'cache unexpectedly accepted after binary contents changed' >&2
    exit 1
fi

rm "$stamp"
if old_terrad_cache_valid "$binary" "$stamp" v4.0.1 "$revision"; then
    echo 'cache unexpectedly accepted without verifiable version metadata' >&2
    exit 1
fi

echo 'old terrad cache checks passed'
