#!/bin/bash

# A cached old binary is reusable only when its build-version stamp matches the
# requested tag and the binary still matches the checksum recorded at build time.
old_terrad_cache_valid() {
    local binary=$1
    local stamp=$2
    local requested_version=$3
    local requested_revision=${4:-}
    [[ "$requested_revision" =~ ^[0-9a-f]{40}$ ]] || return 1
    local stamped_version stamped_revision stamped_sha256 actual_sha256

    [[ -f "$binary" && -f "$stamp" ]] || return 1
    stamped_version=$(sed -n 's/^version=//p' "$stamp")
    stamped_revision=$(sed -n 's/^revision=//p' "$stamp")
    [[ "$stamped_revision" == "$requested_revision" ]] || return 1
    stamped_sha256=$(sed -n 's/^sha256=//p' "$stamp")
    [[ "$stamped_version" == "$requested_version" ]] || return 1
    [[ "$stamped_sha256" =~ ^[[:xdigit:]]{64}$ ]] || return 1
    actual_sha256=$(sha256sum "$binary" | awk '{print $1}') || return 1
    [[ "$actual_sha256" == "$stamped_sha256" ]]
}
