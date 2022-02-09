#!/bin/sh
NODE_ID=4be43aa6f1

SCRIPT_DIR="$(dirname "$(realpath -- "$0")")"

SERVER_CERTIFICATE="$SCRIPT_DIR/certs/AmazonRootCA1.pem"
CLIENT_CERTIFICATE="$SCRIPT_DIR/certs/$NODE_ID-certificate.pem.crt"
CLIENT_PRIVATE_KEY="$SCRIPT_DIR/certs/$NODE_ID-private.pem.key"

go run "$SCRIPT_DIR/cmd/agent/agent.go" \
    -v \
    --nodeId "$NODE_ID" \
    --broker tls://asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com:8883 \
    --rootcert "$SERVER_CERTIFICATE" \
    --cert "$CLIENT_CERTIFICATE" \
    --key "$CLIENT_PRIVATE_KEY" \
    --subClientId nodes/awsdemo \
    --pubClientId manager/awsdemo \
    --publish status \
    --heartbeat 10 \
    --loglevel debug \
    --config "$SCRIPT_DIR/nodeconfig.json"
