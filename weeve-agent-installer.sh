#!/bin/sh

log() {
  # logger
  echo '[' "$(date +"%Y-%m-%d %T")" ']:' INFO "$@" | tee -a "$LOG_FILE"
}

validate_token_file(){
  # looking for the file containing the github access token to download the dependencies
  if [ -z "$TOKEN_FILE" ]; then
    log Missing argument: tokenpath
    log -----------------------------------------------------------------------
    log Make sure you have a file containing the token
    log Follow the steps :
    log 1. Create a hidden file
    log 2. Paste the Github Personal Access Token into the above mentioned file
    log For more info checkout the README
    log ------------------------------------------------------------------------
    exit 1
  fi
}

get_token(){
  if [ -f "$TOKEN_FILE" ];then
    log Reading the access key ...
    ACCESS_KEY=$(cat "$TOKEN_FILE")
  else
    log The required file containing the token not found in the path: "$TOKEN_FILE"
    exit 1
  fi
}

get_environment(){
  # reading values from the user
  if [ -z "$ENV" ]; then
    read -r -p "Enter the environment in which the node is to be registered: " ENV
  fi
}

get_nodename(){
  if [ -z "$NODE_NAME" ]; then
    read -r -p "Enter node name: " NODE_NAME
  fi
}

get_release(){
  if [ -z "$AGENT_RELEASE" ]; then
    read -r -p "Select release [stable, dev]: " AGENT_RELEASE
  fi
}

get_test(){
  if [ -z "$BUILD_LOCAL" ]; then
    BUILD_LOCAL="false"
  fi
}

check_for_agent(){
  # looking for existing agent instance
  if [ -d "$WEEVE_AGENT_DIRECTORY" ] || [ -f "$SERVICE_FILE" ] || [ -f "$ARGUMENTS_FILE" ]; then
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
  if RESULT=$(go build -o ./weeve-agent/test-agent cmd/agent/agent.go 2>&1); then
    log built weeve-agent binary for testing
    chmod u+x ./weeve-agent/test-agent
    log Changed file permission
    BINARY_NAME="test-agent"
  else
    log Error occured while building binary for testing: "$RESULT"
    exit 1
  fi
}

get_bucket_name(){
    if [ "$AGENT_RELEASE" = "stable" ]; then
      S3_BUCKET="weeve-agent"
    elif [ "$AGENT_RELEASE" = "dev" ]; then
      S3_BUCKET="weeve-agent-legacy-dev"
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
  && cd ./weeve-agent \
  && wget http://"$S3_BUCKET".s3.amazonaws.com/"$BINARY_NAME" 2>&1); then
    log Weeve-agent binary downloaded
    chmod u+x ./weeve-agent/"$BINARY_NAME"
    log Changed file permission
  else
    log Error while downloading the executable: "$RESULT"
    CLEANUP="true"
    exit 1
  fi
}

download_dependencies(){
  log Downloading the dependencies ...
  for DEPENDENCIES in AmazonRootCA1.pem aws"$ENV"-certificate.pem.crt aws"$ENV"-private.pem.key weeve-agent.service weeve-agent.argconf
  do
  if RESULT=$(cd ./weeve-agent \
  && curl -sO https://"$ACCESS_KEY"@raw.githubusercontent.com/weeveiot/weeve-agent-legacy-dependencies/master/"$DEPENDENCIES" 2>&1); then
    log "$DEPENDENCIES" downloaded
  else
    log Error while downloading the dependencies: "$RESULT"
    CLEANUP="ture"
    exit 1
  fi
  done
  log Dependencies downloaded.
}

create_nodeconfig(){
  log Creating the node configuration file ...
  {
    printf "{\n"
    printf "\"RootCertPath\": \"AmazonRootCA1.pem\",\n"
    printf "\"CertPath\": \"aws"$ENV"-certificate.pem.crt\",\n"
    printf "\"KeyPath\": \"aws"$ENV"-private.pem.key\",\n"
    printf "\"NodeId\": \"\",\n"
    printf "\"NodeName\": \"\",\n"
    printf "\"Registered\": false\n"
    printf "}\n"
  } >> ./weeve-agent/nodeconfig.json
}

write_to_argconf(){
  log Appeding the required command line arguments by the agent ...
  {
    printf "ARG_SUB_CLIENT=--subClientId nodes/aws%s\n" "$ENV"
    printf "ARG_PUB_CLIENT=--pubClientId manager/aws%s\n" "$ENV"
    printf "ARG_ROOT_CERT=--rootcert AmazonRootCA1.pem\n"
    printf "ARG_CERT=--cert aws%s-certificate.pem.crt\n" "$ENV"
    printf "ARG_KEY=--key aws%s-private.pem.key\n" "$ENV"
    printf "ARG_NODENAME=--name %s" "$NODE_NAME"
  }  >> ./weeve-agent/weeve-agent.argconf
}

write_to_service(){
  # appending the required strings to the .service to point systemd to the path of the binary and to run it
  # following are the example for the lines appended to weeve-agent.service:
  #   WorkingDirectory=/home/admin/weeve-agent
  #   ExecStart=/home/admin/weeve-agent/weeve-agent-x86_64 $ARG_VERBOSE $ARG_HEARTBEAT $ARG_BROKER $ARG_PUBLISH $ARG_SUB_CLIENT $ARG_PUB_CLIENT $ARG_NODENAME

  # line 1
  WORKING_DIRECTORY="WorkingDirectory=$PWD/weeve-agent"

  # line 2
  BINARY_PATH="ExecStart=$PWD/weeve-agent/$BINARY_NAME"
  ARGUMENTS='$ARG_VERBOSE $ARG_HEARTBEAT $ARG_BROKER $ARG_PUBLISH $ARG_SUB_CLIENT $ARG_PUB_CLIENT $ARG_NODENAME'
  EXECUTE_BINARY="$BINARY_PATH $ARGUMENTS"

  log Adding the binary path to service file ...
  {
    printf "%s\n" "$WORKING_DIRECTORY"
    printf "%s\n" "$EXECUTE_BINARY"
  } >> ./weeve-agent/weeve-agent.service
}

start_service(){
  log Starting the service ...

  # moving .service and .argconf to systemd and starting the service
  if RESULT=$(mv weeve-agent/weeve-agent.service /lib/systemd/system/ \
  && mv weeve-agent/weeve-agent.argconf /lib/systemd/system/ \
  && systemctl enable weeve-agent \
  && systemctl start weeve-agent 2>&1); then
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
  timeout 10s tail -f ./weeve-agent/Weeve_Agent.log | sed '/ON connect >> connected >> registered : true/ q'
}

cleanup() {
  # function to clean-up the contents on failure at any point
  # note that this function will be called even at successful ending of the script hence the conditional execution using variable CLEANUP
  if [ "$CLEANUP" = "true" ]; then

    log cleaning up the contents ...

    if RESULT=$(systemctl is-active weeve-agent 2>&1); then
      systemctl stop weeve-agent
      systemctl daemon-reload
      log weeve-agent service stopped
    else
      log weeve-agent service not running
    fi

    if [ -f "$SERVICE_FILE" ]; then
      rm "$SERVICE_FILE"
      log "$SERVICE_FILE" removed
    else
      log "$SERVICE_FILE" doesnt exists
    fi

    if [ -f "$ARGUMENTS_FILE" ]; then
      rm "$ARGUMENTS_FILE"
      log "$ARGUMENTS_FILE" removed
    else
      log "$ARGUMENTS_FILE" doesnt exists
    fi

    if [ -d "$WEEVE_AGENT_DIRECTORY" ] ; then
      rm -r "$WEEVE_AGENT_DIRECTORY"
      log "$WEEVE_AGENT_DIRECTORY" removed
    else
      log "$WEEVE_AGENT_DIRECTORY" doesnt exists
    fi

  fi
}

# if in case the user have deleted the weeve-agent.service and did not reload the systemd daemon
systemctl daemon-reload

# Delcaring and defining variables
LOG_FILE=installer.log

WEEVE_AGENT_DIRECTORY="$PWD"/weeve-agent

SERVICE_FILE=/lib/systemd/system/weeve-agent.service

ARGUMENTS_FILE=/lib/systemd/system/weeve-agent.argconf

ACCESS_KEY=""

BUILD_LOCAL=""

S3_BUCKET=""

BINARY_NAME=""

CLEANUP="false"

trap cleanup EXIT

# read command line arguments
for ARGUMENT in "$@"
do
  KEY=$(echo "$ARGUMENT" | cut --fields 1 --delimiter='=')
  VALUE=$(echo "$ARGUMENT" | cut --fields 2 --delimiter='=')

  case "$KEY" in
    "tokenpath") TOKEN_FILE="$VALUE" ;;
    "environment") ENV="$VALUE" ;;
    "nodename")  NODE_NAME="$VALUE" ;;
    "release") AGENT_RELEASE="$VALUE" ;;
    "test") BUILD_LOCAL="$VALUE" ;;
    *)
  esac
done

validate_token_file

get_token

get_environment

get_nodename

get_test

log All arguments are set
log Environment is set to "$ENV"
log Name of the node is set to "$NODE_NAME"
log Test mode is set to "$BUILD_LOCAL"

check_for_agent

validating_docker

if [ "$BUILD_LOCAL" = "true" ]; then
  build_test_binary
else
  get_release

  get_bucket_name

  download_binary
fi

download_dependencies

create_nodeconfig

write_to_argconf

write_to_service

start_service

tail_agent_log