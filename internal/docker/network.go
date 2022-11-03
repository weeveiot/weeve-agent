//go:build !secunet

package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	linq "github.com/ahmetb/go-linq/v3"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/model"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

// Network name constraints
const manifestNamelength = 11
const indexLength = 3
const maxNetworkIndex = 999

func readAllNetworks() ([]types.NetworkResource, error) {
	log.Debug("Docker_container -> readAllNetworks")

	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, traceutility.FuncTrace())
	}

	return networks, nil
}

func ReadDataServiceNetworks(manifestUniqueID model.ManifestUniqueID) ([]types.NetworkResource, error) {
	log.Debug("Docker_container -> ReadDataServiceNetworks")

	filter := filters.NewArgs()
	filter.Add("label", "manifestName="+manifestUniqueID.ManifestName)
	filter.Add("label", "versionNumber="+manifestUniqueID.VersionNumber)
	options := types.NetworkListOptions{Filters: filter}

	networks, err := dockerClient.NetworkList(ctx, options)
	if err != nil {
		return nil, errors.Wrap(err, traceutility.FuncTrace())
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
		return "", errors.Wrap(err, traceutility.FuncTrace())
	}
	if maxCount < maxNetworkIndex {
		newCount = maxCount + 1
	} else {
		newCount, err = getLowestAvailableNetworkCount()
		if err != nil {
			return "", errors.Wrap(err, traceutility.FuncTrace())
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
		return "", errors.Wrap(err, traceutility.FuncTrace())
	}
	if networkName == "" {
		return "", errors.New("failed to generate network name")
	}

	_, err = dockerClient.NetworkCreate(context.Background(), networkName, networkCreateOptions)
	if err != nil {
		return networkName, errors.Wrap(err, traceutility.FuncTrace())
	}

	return networkName, nil
}

func NetworkPrune(manifestUniqueID model.ManifestUniqueID) error {
	filter := filters.NewArgs()
	filter.Add("label", "manifestName="+manifestUniqueID.ManifestName)
	filter.Add("label", "versionNumber="+manifestUniqueID.VersionNumber)

	pruneReport, err := dockerClient.NetworksPrune(ctx, filter)
	if err != nil {
		return errors.Wrap(err, traceutility.FuncTrace())
	}
	log.Info("Pruned networks: ", pruneReport.NetworksDeleted)
	return nil
}

func getLastCreatedNetworkCount() (int, error) {
	maxCount := 0

	counts, err := getExistingNetworkCounts()
	if err != nil {
		return 0, errors.Wrap(err, traceutility.FuncTrace())
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
		return 0, errors.Wrap(err, traceutility.FuncTrace())
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
		return nil, errors.Wrap(err, traceutility.FuncTrace())
	}
	linq.From(networks).Select(func(c interface{}) interface{} {
		nm := c.(types.NetworkResource).Name
		nm = nm[len(nm)-indexLength:]
		count, _ := strconv.Atoi(nm)
		return count
	}).ToSlice(&counts)
	return counts, nil
}
