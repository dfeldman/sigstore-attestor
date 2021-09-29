package main

// This is the sigstore attestor
// Most of it is copied from the Docker attestor, just with additional Sigstore
// functionality and tweaked to work as an external rather than internal plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl"
	workloadattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/agent/workloadattestor/v1"
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"
	"github.com/spiffe/spire/pkg/agent/common/cgroups"
	"github.com/spiffe/spire/pkg/agent/plugin/workloadattestor/docker/cgroup"

	"github.com/spiffe/spire-plugin-sdk/pluginmain"
	//"github.com/spiffe/spire-plugin-sdk/pluginsdk"
)

const (
	pluginName         = "sigstore"
	subselectorLabel   = "sigstore_label"
	subselectorImageID = "sigstore_image_id"
	subselectorEnv     = "sigstore_env"
)

// Docker is a subset of the docker client functionality, useful for mocking.
type Docker interface {
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}

type Plugin struct {
	workloadattestorv1.UnsafeWorkloadAttestorServer
	configv1.UnsafeConfigServer

	log     hclog.Logger
	fs      cgroups.FileSystem
	retryer *retryer

	mtx               sync.RWMutex
	containerIDFinder cgroup.ContainerIDFinder
	docker            Docker
}

// unused
func New() *Plugin {
	return &Plugin{
		fs:      cgroups.OSFileSystem{},
		retryer: newRetryer(),
	}
}

type dockerPluginConfig struct {
	// DockerSocketPath is the location of the docker daemon socket (default: "unix:///var/run/docker.sock" on unix).
	DockerSocketPath string `hcl:"docker_socket_path"`
	// DockerVersion is the API version of the docker daemon. If not specified, the version is negotiated by the client.
	DockerVersion string `hcl:"docker_version"`
	// ContainerIDCGroupMatchers is a list of patterns used to discover container IDs from cgroup entries.
	// See the documentation for cgroup.NewContainerIDFinder in the cgroup subpackage for more information.
	ContainerIDCGroupMatchers []string `hcl:"container_id_cgroup_matchers"`

	Registry string `hcl:docker_registry`

	PathToCosign string `hcl:path_to_cosign`
}

func (p *Plugin) SetLogger(log hclog.Logger) {
	p.log = log
}

func (p *Plugin) Attest(ctx context.Context, req *workloadattestorv1.AttestRequest) (*workloadattestorv1.AttestResponse, error) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if p.containerIDFinder == nil {
		p.log.Info("Refusing to attest becuase Configure never called")
		return nil, nil
	}
	p.log.Info("DJF It's logging! %i ", req.Pid)
	if req == nil {
		p.log.Info("request is nil")
	}
	if p.fs == nil {
		p.log.Info("fs is nil")
	}
	cgroupList, err := cgroups.GetCgroups(req.Pid, cgroups.OSFileSystem{})
	if err != nil {
		return nil, err
	}
	p.log.Info("DJF 2")

	containerID, err := getContainerIDFromCGroups(p.containerIDFinder, cgroupList)
	switch {
	case err != nil:
		p.log.Error("Unable to get container data for cgroup list")
		return nil, err
	case containerID == "":
		p.log.Error("Unable to get container data for cgroup list (non docker")
		// Not a docker workload. Nothing more to do.
		return &workloadattestorv1.AttestResponse{}, nil
	}
	p.log.Info("DJF 3")

	if p.retryer == nil {
		p.log.Error("Retryer is nil")
	}
	if p.docker == nil {
		p.log.Error("docker is nil")
	}

	var container types.ContainerJSON
	err = p.retryer.Retry(ctx, func() error {
		container, err = p.docker.ContainerInspect(ctx, containerID)
		if err != nil {
			p.log.Error("Unable to find container details")
			return err
		}
		p.log.Error("Retry")
		return nil
	})
	p.log.Info("DJF 4")

	if err != nil {
		return nil, err
	}
	p.log.Info("DJF HEY the image name is %v", container.Config.Image)
	return &workloadattestorv1.AttestResponse{
		SelectorValues: p.getSelectorValuesFromCosign(container.Config),
	}, nil
}

func (p *Plugin) getSelectorValuesFromCosign(cfg *container.Config) []string {
	var selectorValues []string
	if cfg.Image == "" {
		p.log.Error("Image ID is not available. Unable to do sigstore attestation.")
	} else {
		selectorValues = append(selectorValues, fmt.Sprintf("%s:%s", subselectorImageID, cfg.Image))
	}

	registry := "docker.io"
	docker_full_path := registry + "/" + cfg.Image
	cmd := exec.Command("/home/dfeldman/work/sigstore-attestor/bin/cosign", "verify", docker_full_path)
	cmd.Env = append(cmd.Env, "COSIGN_EXPERIMENTAL=1")
	stdout, err := cmd.Output()

	if err != nil {
		p.log.Error("Can't run cosign", "error", hclog.Fmt("%v", string(stdout)))
	}

	p.log.Info(string(stdout))

	parsedOutput, err := p.cosignOutputToSubject(stdout)
	if err == nil {
		selectorValues = append(selectorValues, parsedOutput)
	}

	return selectorValues
}

type CosignOutputItem struct {
	Optional CosignOptionalItem `json:"optional"`
}

type CosignOptionalItem struct {
	Subject []string `json:"subject"`
}

func (p *Plugin) cosignOutputToSubject(output []byte) (string, error) {
	var cosignOutputParsed []CosignOutputItem
	err := json.Unmarshal(output, &cosignOutputParsed)
	p.log.Info("cosign output", "out", string(output))
	if err != nil {
		p.log.Error("Error parsing cosign output", "err", err)
		return "", nil
	}
	p.log.Info("Parsed output", "out", cosignOutputParsed)
	if len(cosignOutputParsed) == 0 {
		p.log.Error("Cosign did not return a valid output signature")
		return "", nil
	}
	if len(cosignOutputParsed[0].Optional.Subject) == 0 {
		p.log.Error("Cosign did not return a signature subject")
		return "", nil
	}
	p.log.Info("Identified subject from sigstore", "subj", cosignOutputParsed[0].Optional.Subject[0])
	return "subject:" + cosignOutputParsed[0].Optional.Subject[0], nil
}

func (p *Plugin) Configure(ctx context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	var err error
	p.log.Info("DJF CONFIGURATION")

	config := &dockerPluginConfig{}
	if err = hcl.Decode(config, req.HclConfiguration); err != nil {
		return nil, err
	}

	var opts []dockerclient.Opt
	if config.DockerSocketPath != "" {
		opts = append(opts, dockerclient.WithHost(config.DockerSocketPath))
	}
	switch {
	case config.DockerVersion != "":
		opts = append(opts, dockerclient.WithVersion(config.DockerVersion))
	default:
		opts = append(opts, dockerclient.WithAPIVersionNegotiation())
	}

	docker, err := dockerclient.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}

	var containerIDFinder cgroup.ContainerIDFinder = &defaultContainerIDFinder{}
	if len(config.ContainerIDCGroupMatchers) > 0 {
		containerIDFinder, err = cgroup.NewContainerIDFinder(config.ContainerIDCGroupMatchers)
		if err != nil {
			return nil, err
		}
	}

	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.docker = docker
	p.containerIDFinder = containerIDFinder
	p.retryer = newRetryer()

	return &configv1.ConfigureResponse{}, nil
}

// getContainerIDFromCGroups returns the container ID from a set of cgroups
// using the given finder. The container ID found on each cgroup path (if any)
// must be consistent. If no container ID is found among the cgroups, i.e.,
// this isn't a docker workload, the function returns an empty string. If more
// than one container ID is found, or the "found" container ID is blank, the
// function will fail.
func getContainerIDFromCGroups(finder cgroup.ContainerIDFinder, cgroups []cgroups.Cgroup) (string, error) {
	var hasDockerEntries bool
	var containerID string
	for _, cgroup := range cgroups {
		candidate, ok := finder.FindContainerID(cgroup.GroupPath)
		if !ok {
			continue
		}

		hasDockerEntries = true

		switch {
		case containerID == "":
			// This is the first container ID found so far.
			containerID = candidate
		case containerID != candidate:
			// More than one container ID found in the cgroups.
			return "", fmt.Errorf("workloadattestor/docker: multiple container IDs found in cgroups (%s, %s)",
				containerID, candidate)
		}
	}

	switch {
	case !hasDockerEntries:
		// Not a docker workload. Since it is expected that non-docker workloads will call the
		// workload API, it is fine to return a response without any selectors.
		return "", nil
	case containerID == "":
		// The "finder" found a container ID, but it was blank. This is a
		// defensive measure against bad matcher patterns and shouldn't
		// be possible with the default finder.
		return "", errors.New("workloadattestor/docker: a pattern matched, but no container id was found")
	default:
		return containerID, nil
	}
}

// dockerCGroupRE matches cgroup paths that have the following properties.
// 1) `\bdocker\b` the whole word docker
// 2) `.+` followed by one or more characters (which will start on a word boundary due to #1)
// 3) `\b([[:xdigit:]][64])\b` followed by a 64 hex-character container id on word boundary
//
// The "docker" prefix and 64-hex character container id can be anywhere in the path. The only
// requirement is that the docker prefix comes before the id.
var dockerCGroupRE = regexp.MustCompile(`\bdocker\b.+\b([[:xdigit:]]{64})\b`)

type defaultContainerIDFinder struct{}

// FindContainerID returns the container ID in the given cgroup path. The cgroup
// path must have the whole word "docker" at some point in the path followed
// at some point by a 64 hex-character container ID. If the cgroup path does
// not match the above description, the method returns false.
func (f *defaultContainerIDFinder) FindContainerID(cgroupPath string) (string, bool) {
	m := dockerCGroupRE.FindStringSubmatch(cgroupPath)
	if m != nil {
		return m[1], true
	}
	return "", false
}

func main() {
	plugin := new(Plugin)
	// Serve the plugin. This function call will not return. If there is a
	// failure to serve, the process will exit with a non-zero exit code.
	pluginmain.Serve(
		workloadattestorv1.WorkloadAttestorPluginServer(plugin),
		// TODO: Remove if no configuration is required
		configv1.ConfigServiceServer(plugin),
	)
}
