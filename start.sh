NODE_ID=7ced3826-7738-4c9f-84b8-9302cb436e89


SERVER_CERTIFICATE=./AmazonRootCA1.pem
CLIENT_CERTIFICATE=./$NODE_ID-certificate.pem.crt
CLIENT_PRIVATE_KEY=./$NODE_ID-private.pem.key

sudo ./agent -v \
    --nodeId $NODE_ID \ # ID of this node \
    --broker tls://asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com:8883 \ # Broker to connect to \
    --rootcert $SERVER_CERTIFICATE \ #\
    --cert $CLIENT_CERTIFICATE \ #\
    --key $CLIENT_PRIVATE_KEY \ #\
    --subClientId nodes/awsdemo \ # Subscriber ClientId \
    --pubClientId manager/awsdemo \ # Publisher ClientId \
    --publish status \ # Topic bame for publishing status messages \
    --heartbeat 10