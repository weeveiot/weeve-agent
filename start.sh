#!/usr/bin/bash
NODE_ID=4be43aa6f1

# source: https://stackoverflow.com/questions/59895/how-can-i-get-the-source-directory-of-a-bash-script-from-within-the-script-itsel
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

SERVER_CERTIFICATE=$SCRIPT_DIR/certs/AmazonRootCA1.pem
CLIENT_CERTIFICATE=$SCRIPT_DIR/certs/$NODE_ID-certificate.pem.crt
CLIENT_PRIVATE_KEY=$SCRIPT_DIR/certs/$NODE_ID-private.pem.key

go run $SCRIPT_DIR/cmd/agent/agent.go \
    -v \ # verbose \
    --nodeId $NODE_ID \ # ID of this node \
    --broker tls://asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com:8883 \ # Broker to connect to \
    --rootcert $SERVER_CERTIFICATE \ #\
    --cert $CLIENT_CERTIFICATE \ #\
    --key $CLIENT_PRIVATE_KEY \ #\
    --subClientId nodes/awsdemo \ # Subscriber ClientId \
    --pubClientId manager/awsdemo \ # Publisher ClientId \
    --publish status \ # Topic bame for publishing status messages \
    --heartbeat 10 \
    --loglevel debug \
    --config $SCRIPT_DIR/nodeconfig.json
