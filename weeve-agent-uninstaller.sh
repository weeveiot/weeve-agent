#!/bin/sh

# logger
log() {
  echo '[' "$(date +"%Y-%m-%d %T")" ']:' INFO "$@"
}

log Detecting the OS of the machine ...
OS=$(uname -s)
log Detected OS: "$OS"

if [ "$OS" = "Linux" ]; then
  # if in case the user have deleted the weeve-agent.service and did not reload the systemd daemon
  sudo systemctl daemon-reload
fi

WEEVE_AGENT_DIR="$PWD/weeve-agent"

SERVICE_FILE=/lib/systemd/system/weeve-agent.service

#* NOTE: tailing will be much faster than the appearing of the lastest weeve-agent logs
if RESULT=$(sudo systemctl stop weeve-agent \
  && printf " --disconnect" >> "$SERVICE_FILE" \
  && sudo systemctl daemon-reload \
  && sudo systemctl start weeve-agent \
  && tail -f "$WEEVE_AGENT_DIR"/Weeve_Agent.log | sed '/weeve agent disconnected/ q' 2>&1); then
  log weeve-agent disconnected
else
  log Error while restarting weeve-agent for disconnection: "$RESULT"
fi

if [ "$OS" = "Linux" ]; then
  if RESULT=$(sudo systemctl stop weeve-agent \
    && sudo systemctl daemon-reload 2>&1); then
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
fi

if [ -d "$WEEVE_AGENT_DIR" ] ; then
  sudo rm -r "$WEEVE_AGENT_DIR"
  log "$WEEVE_AGENT_DIR" removed
else
  log "$WEEVE_AGENT_DIR" doesnt exists
fi

log weeve-agent uninstalled