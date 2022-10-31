//go:build secunet

package docker

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ahmetb/go-linq/v3"
	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var existingNetworks = make(map[string]string)

// Network name constraints
const manifestNamelength = 11
const indexLength = 3
const maxNetworkIndex = 999

func makeNetworkName(name string) string {
	format := "%s_%0" + strconv.Itoa(indexLength) + "d"

	// Prune the name if necessary
	if name == "" {
		return ""
	} else if len(name) > manifestNamelength {
		name = name[:manifestNamelength]
	}

	// Get new network count
	var newCount int
	maxCount := getLastCreatedNetworkCount()
	if maxCount < maxNetworkIndex {
		newCount = maxCount + 1
	} else {
		newCount = getLowestAvailableNetworkCount()
		if newCount < 0 { // no available network count found
			log.Warning("Number of data services limit is exceeded")
			return ""
		}
	}

	// Generate next network name
	networkName := fmt.Sprintf(format, name, newCount)

	return strings.ReplaceAll(networkName, " ", "")
}

func readAllNetworks() []types.NetworkResource {
	log.Debug("Docker_container -> readAllNetworks")

	var networks []types.NetworkResource

	for _, networkName := range existingNetworks {
		networks = append(networks, types.NetworkResource{
			Name: networkName,
		})
	}

	return networks
}

func ReadDataServiceNetworks(manifestUniqueID model.ManifestUniqueID) ([]types.NetworkResource, error) {
	key := manifestUniqueID.ManifestName + manifestUniqueID.VersionNumber
	networkName := existingNetworks[key]

	if networkName == "" {
		return nil, nil
	} else {
		networks := []types.NetworkResource{
			{
				Name: networkName,
			},
		}
		return networks, nil
	}
}

func CreateNetwork(name string, labels map[string]string) (string, error) {
	networkName := makeNetworkName(name)
	if networkName == "" {
		return "", errors.New("failed to generate network name")
	}

	key := labels["manifestName"] + labels["versionNumber"]
	existingNetworks[key] = networkName
	return networkName, nil
}

func NetworkPrune(manifestUniqueID model.ManifestUniqueID) error {
	key := manifestUniqueID.ManifestName + manifestUniqueID.VersionNumber
	delete(existingNetworks, key)
	return nil
}

func getLastCreatedNetworkCount() int {
	maxCount := 0

	counts := getExistingNetworkCounts()

	for _, e := range counts {
		if e > maxCount {
			maxCount = e
		}
	}

	return maxCount
}

func getLowestAvailableNetworkCount() int {
	counts := getExistingNetworkCounts()

	// find lowest available network count
	for minAvailCount := 0; minAvailCount < maxNetworkIndex; minAvailCount++ {
		available := true
		for _, existingCount := range counts {
			if minAvailCount == existingCount {
				available = false
				break
			}
		}
		if available {
			return minAvailCount
		}
	}

	// no available count found
	return -1
}

func getExistingNetworkCounts() []int {
	var counts []int
	networks := readAllNetworks()
	linq.From(networks).Select(func(c interface{}) interface{} {
		nm := c.(types.NetworkResource).Name
		nm = nm[len(nm)-indexLength:]
		count, _ := strconv.Atoi(nm)
		return count
	}).ToSlice(&counts)
	return counts
}
