package main

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func main()  {
	var imageName string
	imageName = "grafana/grafana"
    ctx := context.Background()
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        log.Error(err)
		panic(err)
    }

    log.Info("\t\tPulling image " + imageName)
    out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
    if err != nil {
		log.Error(err)
		panic(err)
	}
	// Doesn't work!
    // defer out.Close()

	// Pollutes the STDOUT
	// io.Copy(os.Stdout, out)

	// Handle the output and block until done!
	io.Copy(ioutil.Discard, out)

    log.Info("Pulled image " + imageName + " into host")
}