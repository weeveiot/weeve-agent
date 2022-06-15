#!/bin/sh

# logger
log() {
  echo '[' "$(date +"%Y-%m-%d %T")" ']:' INFO "$@"
}

sudo systemctl daemon-reload

CURRENT_DIRECTORY=$(pwd)
WEEVE_AGENT_DIRECTORY="$CURRENT_DIRECTORY"/weeve-agent

SERVICE_FILE=/lib/systemd/system/weeve-agent.service

if RESULT=$(systemctl is-active weeve-agent 2>&1); then
sudo systemctl stop weeve-agent
sudo systemctl daemon-reload
log weeve-agent service stopped
else
log weeve-agent service not running
fi

if [ -f "$SERVICE_FILE" ]; then
sudo rm "$SERVICE_FILE"
log "$SERVICE_FILE" removed
else
log "$SERVICE_FILE" doesnt exists
fi

if [ -d "$WEEVE_AGENT_DIRECTORY" ] ; then
rm -r "$WEEVE_AGENT_DIRECTORY"
log "$WEEVE_AGENT_DIRECTORY" removed
else
log "$WEEVE_AGENT_DIRECTORY" doesnt exists
fi

log done