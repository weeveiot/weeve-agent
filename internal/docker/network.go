//go:build !secunet

package docker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	linq "github.com/ahmetb/go-linq/v3"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	log "github.com/sirupsen/logrus"
)

// Network name constraints
const manifestNamelength = 11
const indexLength = 3
const maxNetworkIndex = 999

func readAllNetworks() ([]types.NetworkResource, error) {
	log.Debug("Docker_container -> readAllNetworks")

	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	return networks, nil
}

func ReadDataServiceNetworks(manifestID string, version string) ([]types.NetworkResource, error) {
	log.Debug("Docker_container -> ReadDataServiceNetworks")

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.NetworkListOptions{Filters: filter}

	networks, err := dockerClient.NetworkList(ctx, options)
	if err != nil {
		return nil, err
	}

	return networks, nil
}

func makeNetworkName(name string) (string, error) {
	format := "%s_%0" + strconv.Itoa(indexLength) + "d"

	// Prune the name if necessary
	if name == "" {
		return "", nil
	} else if len(name) > manifestNamelength {
		name = name[:manifestNamelength]
	}

	// Get new network count
	var newCount int
	maxCount, err := getLastCreatedNetworkCount()
	if err != nil {
		return "", err
	}
	if maxCount < maxNetworkIndex {
		newCount = maxCount + 1
	} else {
		newCount, err = getLowestAvailableNetworkCount()
		if err != nil {
			return "", err
		}
		if newCount < 0 { // no available network count found
			log.Warning("Number of data services limit is exceeded")
			return "", nil
		}
	}

	// Generate next network name
	networkName := fmt.Sprintf(format, name, newCount)

	return strings.ReplaceAll(networkName, " ", ""), nil
}

func CreateNetwork(name string, labels map[string]string) (string, error) {
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	networkCreateOptions.Labels = labels

	networkName, err := makeNetworkName(name)
	if err != nil {
		return "", err
	}
	if networkName == "" {
		return "", errors.New("failed to generate network name")
	}

	_, err = dockerClient.NetworkCreate(context.Background(), networkName, networkCreateOptions)
	if err != nil {
		return networkName, err
	}

	return networkName, nil
}

func NetworkPrune(manifestID string, version string) error {
	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)

	pruneReport, err := dockerClient.NetworksPrune(ctx, filter)
	if err != nil {
		return err
	}
	log.Info("Pruned networks: ", pruneReport.NetworksDeleted)
	return nil
}

func getLastCreatedNetworkCount() (int, error) {
	maxCount := 0

	counts, err := getExistingNetworkCounts()
	if err != nil {
		return 0, err
	}

	for _, e := range counts {
		if e > maxCount {
			maxCount = e
		}
	}

	return maxCount, nil
}

func getLowestAvailableNetworkCount() (int, error) {
	counts, err := getExistingNetworkCounts()
	if err != nil {
		return 0, err
	}

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
			return minAvailCount, nil
		}
	}

	// no available count found
	return -1, nil
}

func getExistingNetworkCounts() ([]int, error) {
	var counts []int
	networks, err := readAllNetworks()
	if err != nil {
		return nil, err
	}
	linq.From(networks).Select(func(c interface{}) interface{} {
		nm := c.(types.NetworkResource).Name
		nm = nm[len(nm)-indexLength:]
		count, _ := strconv.Atoi(nm)
		return count
	}).ToSlice(&counts)
	return counts, nil
}
