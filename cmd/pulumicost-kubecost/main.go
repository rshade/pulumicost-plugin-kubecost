package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rshade/pulumicost-plugin-kubecost/internal/kubecost"
	"github.com/rshade/pulumicost-plugin-kubecost/internal/server"
	"github.com/rshade/pulumicost-plugin-kubecost/pkg/version"
	// TODO: Add when pulumicost-spec is available
	// pbc "github.com/yourorg/pulumicost-spec/sdk/go/proto"
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showVersionFull := flag.Bool("version-full", false, "Show detailed version information")
	flag.Parse()

	// Handle version flags
	if *showVersion {
		_, _ = os.Stdout.WriteString(version.String() + "\n")
		os.Exit(0)
	}
	if *showVersionFull {
		_, _ = os.Stdout.WriteString(version.FullString() + "\n")
		os.Exit(0)
	}

	cfg, err := kubecost.LoadConfigFromEnvOrFile(os.Getenv("KUBECOST_CONFIG"))
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	cli, err := kubecost.NewClient(cubectx(context.Background()), cfg)
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	log.Printf("pulumicost-kubecost starting, %s", version.String())

	// Pulumi-style plugins often use stdin/stdout. For simplicity here, use a TCP loopback.
	// Your plugin host can launch and connect to this ephemeral port; or adapt to stdio transport.
	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	_ = server.NewKubecostServer(cli)
	// TODO: Uncomment when pulumicost-spec protobuf definitions are available
	// kubecostServer.RegisterService(grpcServer)

	log.Printf("listening on %s", lis.Addr().String())
	if serveErr := grpcServer.Serve(lis); serveErr != nil {
		log.Fatalf("serve: %v", serveErr)
	}
}

const defaultTimeoutSeconds = 30

func cubectx(ctx context.Context) context.Context {
	t := defaultTimeoutSeconds * time.Second
	if d := os.Getenv("KUBECOST_TIMEOUT"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			t = parsed
		}
	}
	c, _ := context.WithTimeout(ctx, t) //nolint:govet // cancel not needed for this use case
	return c
}
