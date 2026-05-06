#!/usr/bin/env sh
# set-worker-version.sh - single source of truth for setting the current
# Worker Deployment Version on the mortgage-worker deployment.
#
# Used in two contexts:
#
#   1. Bootstrap (compose.yaml worker-version service): runs with
#      BOOTSTRAP_ONLY=1 so the script is a no-op when a current version is
#      already set. This is what makes `docker compose up` automatically
#      bootstrap v1, while still being safe to re-run after a manual v2
#      promotion (compose up never reverts v2 back to v1).
#
#   2. Promotion / rollback (Makefile set-worker-version target): runs
#      without BOOTSTRAP_ONLY so the requested DEPLOYMENT_VERSION is always
#      applied, even if a different version is currently active. This is
#      what makes `make set-worker-version DEPLOYMENT_VERSION=mortgage-worker-v2`
#      promote v2, and equally allows rolling back to v1.
#
# The bootstrap idempotency guard matches "${WORKER_DEPLOYMENT_NAME}-v" rather
# than a specific build ID. That broader match is intentional: it treats
# *any* current version as "already configured" so a previous manual v2
# promotion is preserved across container restarts. Matching a specific
# build ID would re-assert v1 on every compose up and silently revert
# operator-driven promotions.
#
# POSIX sh is used (no bash) because the temporalio/temporal image is alpine
# based and only ships /bin/sh. `set -eu` is used; pipefail is a bashism and
# is intentionally not enabled.
#
# CLI form: this script uses `temporal worker deployment ...` (space-separated
# subcommand path), which is the form supported by the Temporal CLI bundled in
# the temporalio/temporal image. The older hyphenated form (`worker-deployment`)
# does not exist on this CLI.
#
# Connection: the CLI implicitly reads the TEMPORAL_ADDRESS env var as the
# default for --address, so this script does not pass --address explicitly.
# The compose.yaml worker-version service and the Makefile both export
# TEMPORAL_ADDRESS into the helper container so the CLI can reach the
# Temporal server at temporal:7233.

set -eu

WORKER_DEPLOYMENT_NAME="${WORKER_DEPLOYMENT_NAME:-mortgage-worker}"
DEPLOYMENT_VERSION="${DEPLOYMENT_VERSION:-mortgage-worker-v2}"
RETRIES="${VERSIONING_INIT_RETRIES:-30}"
SLEEP_SECONDS="${VERSIONING_INIT_SLEEP_SECONDS:-2}"
BOOTSTRAP_ONLY="${BOOTSTRAP_ONLY:-0}"

if [ "$BOOTSTRAP_ONLY" = "1" ]; then
  echo "[bootstrap] Initialising worker deployment version..."
else
  echo "[manual] Setting worker deployment version..."
fi

echo "Target deployment: $WORKER_DEPLOYMENT_NAME"
echo "Target version: $DEPLOYMENT_VERSION"

i=1
while [ "${i}" -le "${RETRIES}" ]; do
    if [ "${BOOTSTRAP_ONLY}" = "1" ] || [ "${BOOTSTRAP_ONLY}" = "true" ]; then
        if temporal worker deployment describe \
                --name "${WORKER_DEPLOYMENT_NAME}" 2>/dev/null \
            | grep -E "Current Version" \
            | grep -q "${WORKER_DEPLOYMENT_NAME}-v"; then
            echo "Current Worker Deployment Version is already set; nothing to do."
            exit 0
        fi
    fi

    if temporal worker deployment set-current-version \
            --deployment-name "${WORKER_DEPLOYMENT_NAME}" \
            --build-id "${DEPLOYMENT_VERSION}" \
            --yes; then
        echo "Current version set to '${DEPLOYMENT_VERSION}'."
        echo "New workflow executions will be routed to '${DEPLOYMENT_VERSION}'; in-flight executions remain pinned to the version that started them."
        exit 0
    fi

    echo "Build '${DEPLOYMENT_VERSION}' not yet registered with Temporal (attempt ${i}/${RETRIES}); waiting ${SLEEP_SECONDS}s..."
    sleep "${SLEEP_SECONDS}"
    i=$((i + 1))
done

echo "Failed to set Worker Deployment Version after ${RETRIES} attempts" >&2
exit 1
