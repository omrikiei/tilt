package build

import (
	"flag"
	"io"

	"github.com/windmilleng/tilt/internal/docker"
	"github.com/windmilleng/tilt/pkg/model"
)

func Options(archive io.Reader, args model.DockerBuildArgs) docker.BuildOptions {
	return docker.BuildOptions{
		Context:    archive,
		Dockerfile: "Dockerfile",
		Remove:     shouldRemoveImage(),
		BuildArgs:  manifestBuildArgsToDockerBuildArgs(args),
	}
}

func shouldRemoveImage() bool {
	return flag.Lookup("test.v") != nil
}

func manifestBuildArgsToDockerBuildArgs(args model.DockerBuildArgs) map[string]*string {
	r := make(map[string]*string, len(args))
	for k, a := range args {
		tmp := a
		r[k] = &tmp
	}

	return r
}
