#!/bin/sh

log() {
  # logger
  echo '[' "$(date +"%Y-%m-%d %T")" ']:' INFO "$@" | tee -a "$LOG_FILE"
}

get_config(){
  if [ -z "$CONFIG_FILE" ]; then
    read -r -p "Enter the path to the node configuration JSON file: " CONFIG_FILE
  fi
}

validate_config(){
  if [ -f "$CONFIG_FILE" ];then
    log The node configuration JSON file exists
  else
    log The required file containing the node configurations not found in the path: "$CONFIG_FILE"
    exit 1
  fi
}

get_environment(){
  # reading values from the user
  if [ -z "$ENV" ]; then
    read -r -p "Enter the environment in which the node is to be registered: " ENV
  fi
}

get_test(){
  if [ -z "$BUILD_LOCAL" ]; then
    BUILD_LOCAL="false"
  fi
}

get_broker(){
  if [ -z "$BROKER" ]; then
    BROKER="tls://mapi-dev.weeve.engineering:8883"
  fi
}

get_loglevel(){
  if [ -z "$LOG_LEVEL" ]; then
    LOG_LEVEL="info"
  fi
}

get_heartbeat(){
  if [ -z "$HEARTBEAT" ]; then
    HEARTBEAT="300"
  fi
}

check_for_agent(){
  # looking for existing agent instance
  if [ -d "$WEEVE_AGENT_DIR" ] || [ -f "$SERVICE_FILE" ]; then
    log Detected existing weeve-agent contents!
    read -r -p "Proceeding with the installation will cause REMOVAL of the existing contents of weeve-agent! Do you want to proceed? y/n: " RESPONSE
    if [ "$RESPONSE" = "y" ] || [ "$RESPONSE" = "yes" ]; then
      log Proceeding with the removal of existing weeve-agent contents ...
      CLEANUP="true"
      cleanup
      CLEANUP="false"
    else
      log exiting ...
      exit 0
    fi
  fi
}

validating_docker(){
  log Validating if docker is installed and running ...
  if RESULT=$(systemctl is-active docker 2>&1); then
    log Docker is running.
  else
    log Docker is not running, is docker installed?
    log Error while validating docker: "$RESULT"
    log To install docker, visit https://docs.docker.com/engine/install/
    exit 1
  fi
}

build_test_binary(){
  if RESULT=$(go build -o "$WEEVE_AGENT_DIR"/test-agent cmd/agent/agent.go 2>&1); then
    log built weeve-agent binary for testing
    chmod u+x "$WEEVE_AGENT_DIR"/test-agent
    log Changed file permission
    BINARY_NAME="test-agent"
  else
    log Error occured while building binary for testing: "$RESULT"
    exit 1
  fi
}

download_binary(){
  log Detecting the architecture of the machine ...
  ARCH=$(uname -m)
  log Architecture: "$ARCH"

  # detecting the architecture and downloading the respective weeve-agent binary
  case "$ARCH" in
    "x86_64") BINARY_NAME="weeve-agent-amd64"
    ;;
    "arm" | "armv7l") BINARY_NAME="weeve-agent-arm"
    ;;
    "aarch64" | "aarch64_be" | "armv8b" | "armv8l") BINARY_NAME="weeve-agent-arm64"
    ;;
    *) log Unsupported architecture: "$ARCH"
    exit 1
    ;;
  esac

  if RESULT=$(mkdir weeve-agent \
  && cd "$WEEVE_AGENT_DIR" \
  && wget http://"$S3_BUCKET".s3.amazonaws.com/"$BINARY_NAME" 2>&1); then
    log Weeve-agent binary downloaded
    chmod u+x "$WEEVE_AGENT_DIR"/"$BINARY_NAME"
    log Changed file permission
  else
    log Error while downloading the executable: "$RESULT"
    CLEANUP="true"
    exit 1
  fi
}

download_dependencies(){
  log Downloading the dependencies ...
  for DEPENDENCIES in ca.crt weeve-agent.service
  do
  if RESULT=$(cd "$WEEVE_AGENT_DIR" \
  && wget http://"$S3_BUCKET".s3.amazonaws.com/"$DEPENDENCIES" 2>&1); then
    log "$DEPENDENCIES" downloaded
  else
    log Error while downloading the dependencies: "$RESULT"
    CLEANUP="true"
    exit 1
  fi
  done
  log Dependencies downloaded.
}

write_to_service(){
  # appending the required strings to the .service to point systemd to the path of the binary and to run it
  # following are the example for the lines appended to weeve-agent.service

  BINARY_PATH="$WEEVE_AGENT_DIR/$BINARY_NAME"

  # the CLI arguments for weeve agent
  ARG_STDOUT="--out"
  ARG_BROKER="--broker $BROKER"
  ARG_ROOT_CERT="--rootcert $WEEVE_AGENT_DIR/ca.crt"
  ARG_LOG_LEVEL="--loglevel $LOG_LEVEL"
  ARG_HEARTBEAT="--heartbeat $HEARTBEAT"
  ARG_NODECONFIG="--config $CONFIG_FILE"
  ARGUMENTS="$ARG_STDOUT $ARG_HEARTBEAT $ARG_BROKER $ARG_ROOT_CERT $ARG_LOG_LEVEL $ARG_NODECONFIG"
  EXECUTE_BINARY="$BINARY_PATH $ARGUMENTS"

  log Adding the binary path to service file ...
  {
    printf "WorkingDirectory=%s\n" "$WEEVE_AGENT_DIR"
    printf "ExecStart=%s\n" "$EXECUTE_BINARY"
  } >> "$WEEVE_AGENT_DIR"/weeve-agent.service
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
    log For good measure please check:
    log   1. if the file contains the access token
    log   2. if the access token in github is not expired
    CLEANUP="true"
    exit 1
  fi

  sleep 5
}

tail_agent_log(){
  # parsing the weeve-agent log for to verify if the weeve-agent is registered and connected
  # on successful completion of the script $CLEANUP is set to false to skip the clean-up on exit
  log tailing the weeve-agent logs
  timeout 10s tail -f "$WEEVE_AGENT_DIR"/Weeve_Agent.log | sed '/ON connect >> connected >> registered : true/ q'
}

cleanup() {
  # function to clean-up the contents on failure at any point
  # note that this function will be called even at successful ending of the script hence the conditional execution using variable CLEANUP
  if [ "$CLEANUP" = "true" ]; then

    log cleaning up the contents ...

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

    if [ -d "$WEEVE_AGENT_DIR" ] ; then
      sudo rm -r "$WEEVE_AGENT_DIR"
      log "$WEEVE_AGENT_DIR" removed
    else
      log "$WEEVE_AGENT_DIR" doesnt exists
    fi

  fi
}

# if in case the user have deleted the weeve-agent.service and did not reload the systemd daemon
sudo systemctl daemon-reload

# Delcaring and defining variables
LOG_FILE=installer.log

WEEVE_AGENT_DIR="$PWD/weeve-agent"

SERVICE_FILE=/lib/systemd/system/weeve-agent.service

BUILD_LOCAL=""

S3_BUCKET="weeve-agent-dev-binaries"

BINARY_NAME=""

CLEANUP="false"

trap cleanup EXIT

# read command line arguments
for ARGUMENT in "$@"
do
  KEY=$(echo "$ARGUMENT" | cut --fields 1 --delimiter='=')
  VALUE=$(echo "$ARGUMENT" | cut --fields 2 --delimiter='=')

  case "$KEY" in
    "configpath") CONFIG_FILE="$VALUE" ;;
    "environment") ENV="$VALUE" ;;
    "test") BUILD_LOCAL="$VALUE" ;;
    "broker") BROKER="$VALUE" ;;
    "loglevel") LOG_LEVEL="$VALUE" ;;
    "heartbeat") HEARTBEAT="$VALUE" ;;
    *)
  esac
done

get_config

validate_config

get_environment

get_test

get_broker

get_loglevel

get_heartbeat

log All arguments are set
log Environment is set to "$ENV"
log Test mode is set to "$BUILD_LOCAL"
log Broker is set to "$BROKER"
log Log level is set to "$LOG_LEVEL"
log Heartbeat interval is set to "$HEARTBEAT"

check_for_agent

validating_docker

if [ "$BUILD_LOCAL" = "true" ]; then
  build_test_binary
else
  download_binary
fi

download_dependencies

write_to_service

start_service

tail_agent_log