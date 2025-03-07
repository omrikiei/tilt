package engine

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/windmilleng/tilt/internal/docker"
	"github.com/windmilleng/tilt/internal/store"
	"github.com/windmilleng/tilt/internal/testutils"
	"github.com/windmilleng/tilt/internal/testutils/tempdir"
	"github.com/windmilleng/tilt/internal/tiltfile"
	"github.com/windmilleng/tilt/pkg/model"
)

func TestConfigsController(t *testing.T) {
	f := newCCFixture(t)
	defer f.TearDown()

	state := f.st.LockMutableStateForTesting()
	m := model.Manifest{Name: "foo"}
	mt := store.NewManifestTarget(m)
	state.UpsertManifestTarget(mt)
	state.PendingConfigFileChanges["Tiltfile"] = time.Now()
	state.TiltfilePath = f.JoinPath("Tiltfile")
	f.st.UnlockMutableState()

	a := f.run()

	expected := ConfigsReloadedAction{
		Manifests:  []model.Manifest{{Name: "bar"}},
		FinishTime: f.fc.Times[1],
	}

	assert.Equal(t, expected, a)
}

func (f *ccFixture) run() ConfigsReloadedAction {
	// configs_controller uses state.RelativeTiltfilePath, which is relative to wd
	// sometimes the original directory was invalid (e.g., it was another test's temp dir, which was deleted,
	// but not changed out of), and if it was already invalid, then let's not worry about it.
	origDir, _ := os.Getwd()
	err := os.Chdir(f.Path())
	if err != nil {
		f.T().Fatalf("error changing dir: %v", err)
	}
	defer func() {
		if origDir != "" {
			err = os.Chdir(origDir)
			if err != nil {
				f.T().Fatalf("unable to restore original wd: '%v'", err)
			}
		}
	}()

	f.tfl.Result = tiltfile.TiltfileLoadResult{Manifests: []model.Manifest{{Name: "bar"}}}
	f.st.NotifySubscribers(f.ctx)

	a := store.WaitForAction(f.T(), reflect.TypeOf(ConfigsReloadedAction{}), f.getActions)
	cra, ok := a.(ConfigsReloadedAction)
	if !ok {
		f.T().Fatalf("didn't get an action of type %T", ConfigsReloadedAction{})
	}

	return cra
}

type ccFixture struct {
	*tempdir.TempDirFixture
	ctx        context.Context
	cc         *ConfigsController
	st         *store.Store
	getActions func() []store.Action
	tfl        *tiltfile.FakeTiltfileLoader
	fc         *testutils.FakeClock
}

func newCCFixture(t *testing.T) *ccFixture {
	f := tempdir.NewTempDirFixture(t)
	st, getActions := store.NewStoreForTesting()
	tfl := tiltfile.NewFakeTiltfileLoader()
	cc := NewConfigsController(tfl, docker.NewFakeClient())
	fc := testutils.NewRandomFakeClock()
	cc.clock = fc.Clock()
	ctx, _, _ := testutils.CtxAndAnalyticsForTest()
	st.AddSubscriber(ctx, cc)
	go st.Loop(ctx)
	return &ccFixture{
		TempDirFixture: f,
		ctx:            ctx,
		cc:             cc,
		st:             st,
		getActions:     getActions,
		tfl:            tfl,
		fc:             fc,
	}
}
