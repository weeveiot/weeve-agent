package constants

var RoleArn string
var NodeId string
var BrokerUrl string
var CertPrefix string
var Deployed []string

const ManifestFile = "manifests.jsonl"
const ManifestLogFile = "manifests_log.jsonl"
const StatusFile = "status.jsonl"

const NodeConfigFile = "nodeconfig.json"
const KeyCertificate = "Certificate"
const KeyPrivateKey = "PrivateKey"
const KeyNodeId = "NodeId"
const KeyAWSRootCert = "AWSRootCert"
