package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DisallowedLayers is the list of hardcoded disallowed image layers.
var DisallowedLayers = map[string]struct{}{
	"sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1": {},
}

func main() {
	name := flag.String("name", "", "image reference name")
	digest := flag.String("digest", "", "image digest")
	flag.Parse()

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read stdin: %v\n", err)
		os.Exit(1)
	}

	var descriptor ocispec.Descriptor
	if err := json.Unmarshal(input, &descriptor); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse JSON: %v\n", err)
		os.Exit(1)
	}

	ctx := namespaces.WithNamespace(context.Background(), "default")
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to containerd: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	image, err := client.ImageService().Get(ctx, *name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get image: %v\n", err)
		os.Exit(1)
	}

	manifest, err := images.Manifest(ctx, client.ContentStore(), image.Target, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get image manifest: %v\n", err)
		os.Exit(1)
	}

	for _, layer := range manifest.Layers {
		if _, disallowed := DisallowedLayers[layer.Digest.String()]; disallowed {
			fmt.Printf("Image layer %s is disallowed\n", layer.Digest)
			os.Exit(1)
		}
	}

	fmt.Printf("Image %s (digest: %s) verified successfully\n", *name, *digest)
	os.Exit(0)
}
