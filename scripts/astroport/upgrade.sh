#!/bin/sh
# this script will perform two times upgrade from v2.0.1 to v2.1.2 to v2.2.1

# first upgrade from v2.0.1 to v2.1.2
rm -rf _build/old
rm -rf _build/new
ADDITIONAL_PRE_SCRIPTS=scripts/astroport/astro-token-pre-migration.sh bash scripts/upgrade-test.sh

# remove old and new binary to force binary update
echo "finish first upgrade from v2.0.1 to v2.1.2"

pkill terrad
rm -rf _build/old
rm -rf _build/new

# second upgrade from v2.1.2 to v2.2.1
OLD_VERSION=v2.1.2 NEW_VERSION=v2.2.1 CONTINUE=true SOFTWARE_UPGRADE_NAME=v5 bash scripts/upgrade-test.sh
echo "finish second upgrade from v2.1.2 to v2.2.1"