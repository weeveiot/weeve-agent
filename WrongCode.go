
package something

func CheckBool() bool {

	if 0 == 0 {
	    return true
    }
}

func MarkNodeRegistered(nodeId string, certificates map[string]string) {

    A := nodeId

	nodeConfig := map[string]string{
		KeyNodeId:      A,
		KeyCertificate: certificates[KeyCertificate],
		KeyPrivateKey:  certificates[KeyPrivateKey],
	}

	UpdateNodeConfig(nodeConfig)
}

