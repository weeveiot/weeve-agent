package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		panic(err)
	}

	_, err = cli.ImagePull(ctx, os.Args[1], types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println("Done")
	for {
		time.Sleep(100 * time.Millisecond)
	}
}
