#!/usr/bin/sh
set -e

cd ..

yarn _get-cloudformation-exports

. ./cloudformation-exports.txt

echo "DeploymentBucket :: "$DeploymentBucket

# yarn deploy-stack --parameter-overrides Stage=Sandbox \
# DeploymentBucket=$DeploymentBucket FilenameID="$FilenameID" \


echo (aws ec2 create-key-pair --key-name edge_node_demo_1) > edge_node_demo_1.pem

aws cloudformation deploy --template-file node_instance.yaml --stack-name weeve-edge-node-demo-1 --parameter-overrides InstanceName=Weeve_Edge_Node_Demo_1 KeyName=edge_node_demo_1 VpcId=vpc-64669319
aws cloudformation delete-stack --stack-name weeve-edge-node-demo-1

