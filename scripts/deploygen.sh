#!/bin/bash
# vim: ai:ts=8:sw=8:noet
set -efCo pipefail
export SHELLOPTS
IFS=$'\t\n'

command -v helm >/dev/null 2>&1 || { echo 'please install helm'; exit 1; }

HELM_CHART_PATH="${HELM_CHART_PATH:-./deploy/kubernetes/helm/sloth}"
[ -z "$HELM_CHART_PATH" ] && echo "HELM_CHART_PATH env is needed" && exit 1;

GEN_PATH="${GEN_PATH:-./deploy/kubernetes/raw}"
[ -z "$GEN_PATH" ] && echo "GEN_PATH env is needed" && exit 1;

mkdir -p "${GEN_PATH}"

echo "[*] Rendering chart without plugins..."
rm "${GEN_PATH}/sloth.yaml"
helm template sloth "${HELM_CHART_PATH}" \
    --namespace "monitoring" \
    --set "commonPlugins.enabled=false" > "${GEN_PATH}/sloth.yaml"

echo "[*] Rendering chart with plugins..."
rm "${GEN_PATH}/sloth-with-common-plugins.yaml"
helm template sloth "${HELM_CHART_PATH}" \
    --namespace "monitoring" > "${GEN_PATH}/sloth-with-common-plugins.yaml"