#!/bin/sh

# logger
log() {
  echo '[' "$(date +"%Y-%m-%d %T")" ']:' INFO "$@"
}

log Detecting the OS of the machine ...
OS=$(uname -s)
log Detected OS: "$OS"

SERVICE_FILE=/lib/systemd/system/weeve-agent.service

# Exctrating the command to run weeve-agent
LINE=$(grep "ExecStart" "$SERVICE_FILE")
COMMAND="sudo ${LINE#ExecStart=} --delete"

WEEVE_AGENT_DIR="$PWD/weeve-agent"
if [ ! -d "$WEEVE_AGENT_DIR" ]; then
  log weeve-agent directory does not exists in the current path
  log please run the script in the path where weeve-agent directory exists
  exit 1
fi

if [ "$OS" = "Linux" ]; then
  # if in case the user have deleted the weeve-agent.service and did not reload the systemd daemon
  sudo systemctl daemon-reload
fi

if [ "$OS" = "Linux" ]; then
  if RESULT=$(systemctl is-active weeve-agent 2>&1); then
    sudo systemctl stop weeve-agent
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

log weeve-agent disconnecting ...
if RESULT=$(cd "$WEEVE_AGENT_DIR" && eval "$COMMAND" 2>&1); then
  log weeve-agent disconnected
else
  log Error while restarting weeve-agent for disconnection: "$RESULT"
fi

if [ -d "$WEEVE_AGENT_DIR" ] ; then
  sudo rm -r "$WEEVE_AGENT_DIR"
  log "$WEEVE_AGENT_DIR" removed
else
  log "$WEEVE_AGENT_DIR" doesnt exists
fi

log Removed weeve-agent contents if any.
