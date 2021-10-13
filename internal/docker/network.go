package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	linq "github.com/ahmetb/go-linq/v3"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func ReadAllNetworks() []types.NetworkResource {
	log.Debug("Docker_container -> ReadAllNetworks")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return nil
	}

	networks, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.Error(err)
	}

	return networks
}

func GetNetworkName(name string) string {
	networkName := ""

	// Initial values
	manifestNamelength := 11
	indexLength := 3
	presidingDigits := "00"
	maxNetworkIndex := 999

	if len(name) <= 0 {
		return ""
	} else if len(name) > manifestNamelength {
		name = name[:manifestNamelength]
	}

	// Get last network count
	networks := ReadAllNetworks()
	if len(networks) > 0 {
		// Generate next network name
		maxCount := GetLastCreatedNetworkCount(networks, indexLength)
		if maxCount < maxNetworkIndex {
			presidingDigits = fmt.Sprint(presidingDigits, maxCount+1)
		} else {
			lowestAvailCount := GetLowestAvailableNetworkCount(networks, maxNetworkIndex, indexLength)
			if lowestAvailCount == 0 {
				log.Warning("Number of data services limit is exceeded")
				return ""
			}

			presidingDigits = fmt.Sprint(presidingDigits, lowestAvailCount)
		}
	}

	networkName = fmt.Sprint(name, "_", presidingDigits[len(presidingDigits)-indexLength:])

	return strings.ReplaceAll(networkName, " ", "")
}

func AttachContainerNetwork(containerID string, networkName string) error {
	log.Debug("Build context and client")
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		panic(err)
	}

	var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", containerID, "to network", networkName)
	return nil
}

func GetLastCreatedNetworkCount(networks []types.NetworkResource, indexLength int) int {
	maxCount := 0

	var counts []int
	linq.From(networks).Select(func(c interface{}) interface{} {
		nm := c.(types.NetworkResource).Name
		nm = nm[len(nm)-indexLength:]
		count, _ := strconv.Atoi(nm)
		return count
	}).ToSlice(&counts)

	if len(counts) > 0 {
		for _, e := range counts {
			if e > maxCount {
				maxCount = e
			}
		}
	}

	return maxCount
}

func GetLowestAvailableNetworkCount(networks []types.NetworkResource, maxNetworkIndex int, indexLength int) int {
	minAvailCount := 0

	var counts []int
	linq.From(networks).Select(func(c interface{}) interface{} {
		nm := c.(types.NetworkResource).Name
		nm = nm[len(nm)-indexLength:]
		count, _ := strconv.Atoi(nm)
		return count
	}).ToSlice(&counts)

	var availCount []int
	for i := 1; i < maxNetworkIndex; i++ {
		linq.From(counts).Where(func(c interface{}) bool {
			return c.(int) == i
		}).Select(func(c interface{}) interface{} {
			return c.(int)
		}).ToSlice(&availCount)

		if len(availCount) == 0 {
			minAvailCount = i
			break
		}
	}

	return minAvailCount
}
