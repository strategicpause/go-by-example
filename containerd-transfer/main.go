package main

import (
	"context"
	"fmt"
	"log"
	"syscall"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/transfer/image"
	"github.com/containerd/containerd/v2/core/transfer/registry"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/platforms"
)

const (
	DefaultSnapshotter = ""
	ImageRef           = "public.ecr.aws/docker/library/redis:alpine"
)

func main() {
	if err := redisExample(); err != nil {
		log.Fatal(err)
	}
}

func redisExample() error {
	// create a new client connected to the default socket path for containerd
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "transfer-example")

	reg, err := registry.NewOCIRegistry(ctx, ImageRef)
	if err != nil {
		return err
	}

	storeOpts := []image.StoreOpt{
		image.WithUnpack(platforms.DefaultSpec(), DefaultSnapshotter),
		image.WithPlatforms(platforms.DefaultSpec()),
	}
	is := image.NewStore(ImageRef, storeOpts...)
	err = client.Transfer(ctx, reg, is)

	if err != nil {
		return err
	}

	i, err := client.ImageService().Get(ctx, ImageRef)
	if err != nil {
		return err
	}
	img := containerd.NewImage(client, i)

	// create a container
	container, err := client.NewContainer(
		ctx,
		"redis-server",
		containerd.WithImage(img),
		containerd.WithNewSnapshot("redis-server-snapshot", img),
		containerd.WithNewSpec(oci.WithImageConfig(img)),
	)
	if err != nil {
		return err
	}
	defer container.Delete(ctx, containerd.WithSnapshotCleanup)

	// create a task from the container
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return err
	}
	defer task.Delete(ctx)

	// make sure we wait before calling start
	exitStatusC, err := task.Wait(ctx)
	if err != nil {
		return err
	}

	// call start on the task to execute the redis server
	if err := task.Start(ctx); err != nil {
		return err
	}

	// sleep for a lil bit to see the logs
	time.Sleep(3 * time.Second)

	// kill the process and get the exit status
	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return err
	}

	// wait for the process to fully exit and print out the exit status

	status := <-exitStatusC
	code, _, err := status.Result()
	if err != nil {
		return err
	}
	fmt.Printf("redis-server exited with status: %d\n", code)

	return nil
}
