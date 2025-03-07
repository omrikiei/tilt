package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/windmilleng/tilt/internal/testutils"

	"github.com/windmilleng/tilt/internal/dockercompose"
	"github.com/windmilleng/tilt/internal/k8s"
	"github.com/windmilleng/tilt/internal/k8s/testyaml"
	"github.com/windmilleng/tilt/internal/tiltfile"
	"github.com/windmilleng/tilt/pkg/model"
)

func TestDown(t *testing.T) {
	f := newDownFixture(t)
	defer f.TearDown()

	f.tfl.Result = tiltfile.TiltfileLoadResult{Manifests: newK8sManifest()}
	err := f.cmd.down(f.ctx, f.deps)
	assert.NoError(t, err)
	assert.Contains(t, f.kCli.DeletedYaml, "sancho")
}

func TestDownK8sFails(t *testing.T) {
	f := newDownFixture(t)
	defer f.TearDown()

	f.tfl.Result = tiltfile.TiltfileLoadResult{Manifests: newK8sManifest()}
	f.kCli.DeleteError = fmt.Errorf("GARBLEGARBLE")
	err := f.cmd.down(f.ctx, f.deps)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "GARBLEGARBLE")
	}
}

func TestDownDCFails(t *testing.T) {
	f := newDownFixture(t)
	defer f.TearDown()

	f.tfl.Result = tiltfile.TiltfileLoadResult{Manifests: newDCManifest()}
	f.dcc.DownError = fmt.Errorf("GARBLEGARBLE")
	err := f.cmd.down(f.ctx, f.deps)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "GARBLEGARBLE")
	}
}

func newK8sManifest() []model.Manifest {
	return []model.Manifest{model.Manifest{Name: "fe"}.WithDeployTarget(model.K8sTarget{YAML: testyaml.SanchoYAML})}
}

func newDCManifest() []model.Manifest {
	return []model.Manifest{model.Manifest{Name: "fe"}.WithDeployTarget(model.DockerComposeTarget{
		Name:        "fe",
		ConfigPaths: []string{"dc.yaml"},
	})}
}

type downFixture struct {
	t      *testing.T
	ctx    context.Context
	cancel func()
	cmd    *downCmd
	deps   DownDeps
	tfl    *tiltfile.FakeTiltfileLoader
	dcc    *dockercompose.FakeDCClient
	kCli   *k8s.FakeK8sClient
}

func newDownFixture(t *testing.T) downFixture {
	ctx, _, _ := testutils.CtxAndAnalyticsForTest()
	ctx, cancel := context.WithCancel(ctx)
	tfl := tiltfile.NewFakeTiltfileLoader()
	dcc := dockercompose.NewFakeDockerComposeClient(t, ctx)
	kCli := k8s.NewFakeK8sClient()
	downDeps := DownDeps{tfl, dcc, kCli}
	return downFixture{
		t:      t,
		ctx:    ctx,
		cancel: cancel,
		cmd:    &downCmd{},
		deps:   downDeps,
		tfl:    tfl,
		dcc:    dcc,
		kCli:   kCli,
	}
}

func (f *downFixture) TearDown() {
	f.cancel()
}
