#!/bin/sh

log() {
  # logger
  echo '[' "$(date +"%Y-%m-%d %T")" ']:' INFO "$@" | sudo tee -a "$LOG_FILE"
}

get_config(){
  if [ -z "$CONFIG_FILE" ]; then
    read -r -p "Enter the path to the node configuration JSON file: " CONFIG_FILE
  fi
}

validate_config(){
  CONFIG_FILE="$(eval echo "$CONFIG_FILE")"
  if [ -f "$CONFIG_FILE" ]; then
    log The node configuration JSON file exists
    CONFIG_FILE="$(cd "$(dirname "$CONFIG_FILE")" || exit; pwd)/$(basename "$CONFIG_FILE")"
  else
    log The required file containing the node configurations not found in the path: "$CONFIG_FILE"
    log "exiting ..."
    exit 1
  fi
}

get_release(){
  while [ "$RELEASE" != "prod" ] && [ "$RELEASE" != "dev" ]; do
    read -r -p "Enter the release type (prod or dev) or specify the test flag: " RELEASE
  done
}

check_for_agent(){
  # looking for existing agent instance
  if [ -d "$WEEVE_AGENT_DIR" ] || [ -f "$SERVICE_FILE" ]; then
    log Detected existing weeve-agent contents!
    read -r -p "Proceeding with the installation will cause REMOVAL of the existing contents of weeve-agent! Do you want to proceed? y/n: " RESPONSE
    if [ "$RESPONSE" = "y" ] || [ "$RESPONSE" = "yes" ]; then
      log Proceeding with the removal of existing weeve-agent contents ...
      if [ -f "$WEEVE_AGENT_DIR/known_manifests.jsonl" ]; then
        # preserve the contents of known_manifests.jsonl
        sudo mv "$WEEVE_AGENT_DIR/known_manifests.jsonl" /tmp/known_manifests.jsonl
        cleanup
        # restore the contents of known_manifests.jsonl
        mkdir -p "$WEEVE_AGENT_DIR"
        sudo mv /tmp/known_manifests.jsonl "$WEEVE_AGENT_DIR/known_manifests.jsonl"
      else
        cleanup
      fi
    else
      log exiting ...
      exit 0
    fi
  fi
}

validating_docker(){
  log Validating if docker is installed and running ...
  if [ "$OS" = "Linux" ]; then
    if RESULT=$(ls /var/run/docker.sock 2>&1); then
      log Docker is running.
    else
      log Docker is not running, is docker installed?
      log Error while validating docker: "$RESULT"
      log To install docker, visit https://docs.docker.com/engine/install/
    log "exiting ..."
      exit 1
    fi
  fi
}

get_bucket_name(){
    if [ "$RELEASE" = "prod" ]; then
      S3_BUCKET="weeve-agent"
    elif [ "$RELEASE" = "dev" ]; then
      S3_BUCKET="weeve-agent-dev"
    fi
}

set_weeve_url(){
    if [ "$RELEASE" = "prod" ]; then
      WEEVE_URL="mapi-$RELEASE.weeve.network"
    elif [ "$RELEASE" = "dev" ]; then
      WEEVE_URL="mapi-$RELEASE.weeve.engineering"
    fi
}

build_test_binary(){
  if RESULT=$(make build 2>&1); then
    log built weeve-agent binary for testing
    mkdir -p "$WEEVE_AGENT_DIR"
    mv bin/weeve-agent "$WEEVE_AGENT_DIR"/test-agent
    chmod u+x "$WEEVE_AGENT_DIR"/test-agent
    log Changed file permission
    BINARY_NAME="test-agent"
  else
    log Error occured while building binary for testing: "$RESULT"
    log "exiting ..."
    exit 1
  fi
}

copy_dependencies(){
  cp weeve-agent.service ca.crt "$WEEVE_AGENT_DIR"
}

download_binary(){
  log Detecting the architecture of the machine ...
  ARCH=$(uname -m)
  log Architecture: "$ARCH"

  case "$ARCH" in
    "x86_64") BINARY_ARCH="amd64"
    ;;
    "arm" | "armv7l") BINARY_ARCH="arm"
    ;;
    "arm64" | "aarch64" | "aarch64_be" | "armv8b" | "armv8l") BINARY_ARCH="arm64"
    ;;
    *) log Unsupported architecture: "$ARCH"
    log "exiting ..."
    exit 1
    ;;
  esac

  case "$OS" in
    "Linux") BINARY_OS="linux"
    ;;
    "Darwin") BINARY_OS="macos"
    ;;
    *) log Unsupported OS: "$OS"
    log "exiting ..."
    exit 1
    ;;
  esac

  # downloading the respective weeve-agent binary
  BINARY_NAME="weeve-agent-$BINARY_OS-$BINARY_ARCH"

  if RESULT=$(mkdir -p "$WEEVE_AGENT_DIR" \
  && cd "$WEEVE_AGENT_DIR" \
  && wget http://"$S3_BUCKET".s3.amazonaws.com/"$BINARY_NAME" 2>&1); then
    log Weeve-agent binary downloaded
    chmod u+x "$WEEVE_AGENT_DIR"/"$BINARY_NAME"
    log Changed file permission
  else
    log Error while downloading the executable: "$RESULT"
    cleanup
    log "exiting ..."
    exit 1
  fi
}

download_dependencies(){
  log Downloading the dependencies ...
  if RESULT=$(cd "$WEEVE_AGENT_DIR" \
  && wget http://"$S3_BUCKET".s3.amazonaws.com/weeve-agent.service 2>&1 \
  && wget https://"$WEEVE_URL"/public/mqtt-ca -O ca.crt 2>&1); then
    log Dependencies downloaded
  else
    log Error while downloading the dependencies: "$RESULT"
    cleanup
    log "exiting ..."
    exit 1
  fi
}

write_to_service(){
  # appending the required strings to the .service to point systemd to the path of the binary and to run it
  # following are the example for the lines appended to weeve-agent.service

  BINARY_PATH="$WEEVE_AGENT_DIR/$BINARY_NAME"
  ARGUMENTS="--out --config $CONFIG_FILE"

  # remove hardcoded parameters
  sed -i '/ConditionPathExists\|WorkingDirectory\|ExecStart/d' "$WEEVE_AGENT_DIR"/weeve-agent.service

  # the CLI arguments for weeve agent
  if [ -n "$BROKER" ]; then
    ARGUMENTS="$ARGUMENTS --broker $BROKER"
    echo "$BROKER" | grep -q 'tls://' && ARGUMENTS="$ARGUMENTS --rootcert $WEEVE_AGENT_DIR/ca.crt" || ARGUMENTS="$ARGUMENTS --notls"
  fi

  if [ -n "$LOG_LEVEL" ]; then
    ARGUMENTS="$ARGUMENTS --loglevel $LOG_LEVEL"
  fi

  if [ -n "$HEARTBEAT" ]; then
    ARGUMENTS="$ARGUMENTS --heartbeat $HEARTBEAT"
  fi
  EXECUTE_BINARY="$BINARY_PATH $ARGUMENTS"

  log Adding the binary path to service file ...
  {
    printf "WorkingDirectory=%s\n" "$WEEVE_AGENT_DIR"
    printf "ExecStart=%s" "$EXECUTE_BINARY"
  } >> "$WEEVE_AGENT_DIR"/weeve-agent.service
}

execute_binary(){
  log Starting the agent binary ...
  cd "$WEEVE_AGENT_DIR"
  eval "$EXECUTE_BINARY"
}

start_service(){
  log Starting the service ...

  # moving .service to systemd and starting the service
  if RESULT=$(sudo mv "$WEEVE_AGENT_DIR"/weeve-agent.service "$SERVICE_FILE" \
  && sudo systemctl enable weeve-agent \
  && sudo systemctl start weeve-agent 2>&1); then
    log Weeve-agent is initiated ...
  else
    log Error while starting the weeve-agent service: "$RESULT"
    cleanup
    log "exiting ..."
    exit 1
  fi

  sleep 5
}

tail_agent_log(){
  # parsing the weeve-agent log to verify if the weeve-agent is registered and connected
  log tailing the weeve-agent logs
  timeout 10s tail -f "$WEEVE_AGENT_DIR"/Weeve_Agent.log | sed '/ON connect >> connected >> registered : true/ q'
}

cleanup() {
  # function to clean-up the contents on failure at any point
  log cleaning up the contents ...

  if [ "$OS" = "Linux" ]; then
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
  fi

  if [ -d "$WEEVE_AGENT_DIR" ] ; then
    sudo rm -r "$WEEVE_AGENT_DIR"
    log "$WEEVE_AGENT_DIR" removed
  else
    log "$WEEVE_AGENT_DIR" doesnt exists
  fi
}

show_help() {
  cat << EOF
Usage: ./weeve-agent-installer.sh [OPTION]...
Download weeve agent, install and configure it.

Options:
  -h, --help                  Display this help message
  --configpath                Path to the JSON file with node configuration
  --release                   Name of platform the node should be registered with [prod, dev]
  --test                      If specified, build the agent from local sources
  --broker                    URL of the MQTT broker to connect
  --loglevel                  Level of log verbosity
  --heartbeat                 Time period between heartbeat messages (sec)

EOF
}

# Delcaring and defining variables
LOG_FILE=installer.log

WEEVE_AGENT_DIR="$PWD/weeve-agent"

SERVICE_FILE=/lib/systemd/system/weeve-agent.service

options=$(getopt -l "help,configpath:,release:,test,broker:,loglevel:,heartbeat:" -- "h" "$@") || {
  show_help
  exit 1
}

eval set -- "$options"

while true; do
  case "$1" in
    -h|--help)
      show_help
      exit 0
      ;;
    --configpath)
      shift
      CONFIG_FILE="$1"
      ;;
    --release)
      shift
      RELEASE="$1"
      ;;
    --test)
      BUILD_LOCAL=true
      ;;
    --broker)
      shift
      BROKER="$1"
      ;;
    --loglevel)
      shift
      LOG_LEVEL="$1"
      ;;
    --heartbeat)
      shift
      HEARTBEAT="$1"
      ;;
    --)
      shift
      break
      ;;
    esac
    shift
done

log Detecting the OS of the machine ...
OS=$(uname -s)
log Detected OS: "$OS"

if [ "$OS" = "Linux" ]; then
  # if in case the user have deleted the weeve-agent.service and did not reload the systemd daemon
  sudo systemctl daemon-reload
fi

get_config

validate_config

if [ -z "$BUILD_LOCAL" ]; then
  get_release

  get_bucket_name
else
  log Building from local sources, setting release to dev
  RELEASE="dev"
fi

set_weeve_url

check_for_agent

validating_docker

if [ -z "$BUILD_LOCAL" ]; then
  download_binary

  download_dependencies
else
  build_test_binary

  copy_dependencies
fi

write_to_service

if [ "$OS" = "Linux" ]; then
  start_service
  tail_agent_log
else
  execute_binary
fi
