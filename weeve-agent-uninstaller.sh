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

AGENT_DISCONNECT_SCRIPT="$WEEVE_AGENT_DIR/disconnect.sh"

SERVICE_FILE=/lib/systemd/system/weeve-agent.service

# Exctrating the command to run weeve-agent and building a script with it
LINE=$(grep "ExecStart" "$SERVICE_FILE")
COMMAND="${LINE#ExecStart=} --disconnect"
printf "#!/bin/sh \n" >> "$AGENT_DISCONNECT_SCRIPT"
printf "%s" "$COMMAND" >> "$AGENT_DISCONNECT_SCRIPT"

if [ "$OS" = "Linux" ]; then
  if RESULT=$(systemctl is-active weeve-agent 2>&1); then
    if RESULT=$(sudo systemctl stop weeve-agent \
      && sudo systemctl daemon-reload 2>&1); then
      log weeve-agent service stopped
    else
      log Error while stopping the weeve-agent service
    fi
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

#* NOTE: tailing will be much faster than the appearing of the lastest weeve-agent logs
if RESULT=$(sh "$AGENT_DISCONNECT_SCRIPT" \
  && tail -f Weeve_Agent.log | sed '/weeve agent disconnected/ q' 2>&1); then
  sudo rm Weeve_Agent.log
  log weeve-agent disconnected
else
  log Error while restarting weeve-agent for disconnection: "$RESULT"
fi

if [ -f "$AGENT_DISCONNECT_SCRIPT" ] ; then
  sudo rm "$AGENT_DISCONNECT_SCRIPT"
  log "$AGENT_DISCONNECT_SCRIPT" removed
else
  log "$AGENT_DISCONNECT_SCRIPT" doesnt exists
fi

if [ -d "$WEEVE_AGENT_DIR" ] ; then
  sudo rm -r "$WEEVE_AGENT_DIR"
  log "$WEEVE_AGENT_DIR" removed
else
  log "$WEEVE_AGENT_DIR" doesnt exists
fi

log weeve-agent uninstalled
