#!/usr/bin/env bash
# Day-to-day operations against an already-deployed stack — status,
# logs, start/stop/restart, down. Not deploy.sh's job (that packages
# and ships a new commit, then starts it) — this only ever touches
# already-running containers, never packages or ships code.
#
# Copied and adapted directly from cinder's/EphemNet's own
# deploy/ops.sh (same server, `bh2`, same convention).
set -euo pipefail

cd "$(dirname "$0")/.."  # repo root
# shellcheck source=deploy/environments.sh
source deploy/environments.sh

usage() {
	cat >&2 <<EOF
Usage: $0 <ssh-target> <deploy-root> <environment> <command> [service]

  <ssh-target> <deploy-root> <environment>   same meaning as deploy.sh's
                                own arguments — the actual remote path
                                used is <deploy-root>/<environment>.
                                <environment> must be one of:
                                $VALID_ENVIRONMENTS

Commands:
  status              docker compose ps
  logs [service]       follow logs (all services, or one: persona-web, persona-caddy)
  start                docker compose up -d — starts already-built images; use deploy.sh to build+start from new code
  stop                 docker compose stop — containers kept, not removed
  restart [service]    restart everything, or just one service
  down                 stop and remove containers — volumes persist (persona-caddy's cert data survives this)

Examples:
  $0 deploy@203.0.113.9 /opt/persona live status
  $0 deploy@203.0.113.9 /opt/persona live logs persona-web
  $0 deploy@203.0.113.9 /opt/persona dev restart persona-caddy
EOF
	exit 1
}

[ $# -ge 4 ] || usage
DEPLOY_SSH_TARGET="$1"
DEPLOY_ROOT="$2"
ENVIRONMENT="$3"
COMMAND="$4"
SERVICE="${5:-}"

validate_environment "$ENVIRONMENT"
DEPLOY_REMOTE_PATH="$DEPLOY_ROOT/$ENVIRONMENT"

# Product-qualified, not bare <environment> — see deploy/deploy.sh's own
# comment for the real cross-repo Compose-project collision this
# avoids on `bh2`.
COMPOSE_PROJECT="persona-$ENVIRONMENT"
COMPOSE="docker compose -p $COMPOSE_PROJECT -f deploy/docker-compose.yml"

# shellcheck disable=SC2029  # DEPLOY_REMOTE_PATH is ours to expand locally, not the remote's
run_remote() {
	ssh "$DEPLOY_SSH_TARGET" "cd '$DEPLOY_REMOTE_PATH' && $1"
}

case "$COMMAND" in
status)
	run_remote "$COMPOSE ps"
	;;
logs)
	# -t allocates a pty so Ctrl-C actually stops the follow locally,
	# rather than leaving `docker compose logs -f` running on the
	# server after this SSH session exits.
	ssh -t "$DEPLOY_SSH_TARGET" "cd '$DEPLOY_REMOTE_PATH' && $COMPOSE logs -f --tail 200 $SERVICE"
	;;
start | up)
	run_remote "$COMPOSE up -d"
	;;
stop)
	run_remote "$COMPOSE stop"
	;;
restart)
	run_remote "$COMPOSE restart $SERVICE"
	;;
down)
	run_remote "$COMPOSE down"
	;;
*)
	usage
	;;
esac
