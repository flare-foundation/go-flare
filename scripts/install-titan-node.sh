#!/usr/bin/env bash
# Titan Network — Ubuntu node installer (Docker + systemd)
# One-liner:
#   curl -fsSL https://raw.githubusercontent.com/pakeku/go-titan/main/scripts/install-titan-node.sh | sudo bash
#
# Uses the pre-built ghcr.io image — no Go/Java build on the host.

set -euo pipefail

REPO_URL="${TITAN_REPO_URL:-https://github.com/pakeku/go-titan.git}"
REPO_BRANCH="${TITAN_REPO_BRANCH:-main}"
INSTALL_DIR="${TITAN_INSTALL_DIR:-/opt/titan-node}"
SERVICE_NAME="${TITAN_SERVICE_NAME:-titan-node}"
COMPOSE_FILE="docker-compose.single-node.yml"

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo "Run as root: sudo bash $0"
	exit 1
fi

echo "==> Installing packages (curl, git, jq, nano)..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y curl git jq nano ca-certificates gnupg

if ! command -v docker >/dev/null 2>&1; then
	echo "==> Installing Docker..."
	curl -fsSL https://get.docker.com | sh
fi

systemctl enable docker
systemctl start docker

if ! docker compose version >/dev/null 2>&1; then
	echo "Docker Compose plugin missing. Re-run after installing docker-compose-plugin."
	exit 1
fi

echo "==> Fetching go-titan (${REPO_BRANCH}) into ${INSTALL_DIR}..."
if [[ -d "${INSTALL_DIR}/.git" ]]; then
	git -C "${INSTALL_DIR}" fetch origin "${REPO_BRANCH}"
	git -C "${INSTALL_DIR}" checkout "${REPO_BRANCH}"
	git -C "${INSTALL_DIR}" pull --ff-only origin "${REPO_BRANCH}"
else
	git clone --branch "${REPO_BRANCH}" --depth 1 "${REPO_URL}" "${INSTALL_DIR}"
fi

cd "${INSTALL_DIR}"

if [[ ! -f .env ]]; then
	cp .env.single-node.example .env
	DETECTED_IP="$(curl -fsSL --max-time 5 https://api.ipify.org 2>/dev/null || true)"
	if [[ -n "${DETECTED_IP}" ]]; then
		sed -i "s/^PUBLIC_IP=$/PUBLIC_IP=${DETECTED_IP}/" .env
		echo "    Pre-filled PUBLIC_IP=${DETECTED_IP} (edit in nano if wrong)"
	fi
fi

echo ""
echo "==> Configure this server in nano (.env)"
echo "    Parent (first server):  TITAN_ROLE=bootstrap, TITAN_AUTOCONFIGURE_BOOTSTRAP=0"
echo "    Joiner (other servers): TITAN_ROLE=joiner, PARENT_HOST=<parent-ip>, TITAN_AUTOCONFIGURE_BOOTSTRAP=1"
echo "                            TITAN_BOOTSTRAP_ENDPOINT=http://<parent-ip>:9650/ext/info"
echo ""
if [[ -t 0 ]]; then
	nano .env
else
	echo "No interactive terminal — edit ${INSTALL_DIR}/.env before starting the service."
fi

echo "==> Pulling node image..."
docker compose -f "${COMPOSE_FILE}" pull

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -qi "Status: active"; then
	echo "==> Opening firewall ports 9650 (API) and 9651 (staking)..."
	ufw allow 9650/tcp comment "Titan API" >/dev/null || true
	ufw allow 9651/tcp comment "Titan staking" >/dev/null || true
fi

echo "==> Installing systemd service (${SERVICE_NAME})..."
cat >"/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Titan Network Node (Docker Compose)
Documentation=https://github.com/pakeku/go-titan/blob/${REPO_BRANCH}/DEPLOY-docker.md
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${INSTALL_DIR}
Environment=COMPOSE_PROJECT_NAME=titan-node
ExecStart=/usr/bin/docker compose -f ${COMPOSE_FILE} up -d --remove-orphans
ExecStop=/usr/bin/docker compose -f ${COMPOSE_FILE} down
ExecReload=/usr/bin/docker compose -f ${COMPOSE_FILE} up -d --remove-orphans
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"
systemctl restart "${SERVICE_NAME}.service"

echo ""
echo "==> Titan node service started (survives SSH disconnect and reboot)."
echo ""
systemctl --no-pager status "${SERVICE_NAME}.service" || true
echo ""
echo "Health (may take ~1 min on first boot):"
echo "  curl -sf http://localhost:9650/ext/health && echo OK"
echo ""
echo "Parent identity (record on bootstrap server):"
echo "  curl -s -X POST http://localhost:9650/ext/info -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"info.getNodeID\"}' | jq -r .result.nodeID"
echo "  curl -s -X POST http://localhost:9650/ext/info -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"info.getNodeIP\"}' | jq -r .result.ip"
echo ""
echo "Manage:"
echo "  sudo systemctl status ${SERVICE_NAME}"
echo "  sudo systemctl restart ${SERVICE_NAME}"
echo "  sudo nano ${INSTALL_DIR}/.env && sudo systemctl restart ${SERVICE_NAME}"
echo "  docker compose -f ${INSTALL_DIR}/${COMPOSE_FILE} logs -f"
echo ""