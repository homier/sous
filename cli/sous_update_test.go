package cli

import (
	"testing"

	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

var updateStateTests = []struct {
	State                *sous.State
	GDM                  sous.Deployments
	DID                  sous.DeploymentID
	ExpectedErr          string
	ExpectedNumManifests int
}{
	{
		State:       sous.NewState(),
		GDM:         sous.NewDeployments(),
		ExpectedErr: "invalid deploy ID (no cluster name)",
	},
	{
		State: sous.NewState(),
		GDM:   sous.NewDeployments(),
		DID: sous.DeploymentID{
			Cluster:    "blah",
			ManifestID: sous.MustParseManifestID("github.com/user/project"),
		},
		ExpectedErr: `cluster "blah" is not described in defs.yaml`,
	},
	{
		State: &sous.State{
			Defs: sous.Defs{Clusters: sous.Clusters{
				"blah": &sous.Cluster{Name: "blah"},
			}},
			Manifests: sous.NewManifests(),
		},
		GDM: sous.NewDeployments(),
		DID: sous.DeploymentID{
			Cluster:    "blah",
			ManifestID: sous.MustParseManifestID("github.com/user/project"),
		},
		ExpectedNumManifests: 1,
	},
}

func TestUpdateState(t *testing.T) {
	for _, test := range updateStateTests {
		sid := sous.MustNewSourceID(test.DID.ManifestID.Source.Repo, test.DID.ManifestID.Source.Dir, "1.0.0")
		err := updateState(test.State, test.GDM, sid, test.DID)
		if err != nil {
			if test.ExpectedErr == "" {
				t.Error(err)
				continue
			}
			errStr := err.Error()
			if errStr != test.ExpectedErr {
				t.Errorf("got error %q; want %q", errStr, test.ExpectedErr)
			}
			continue
		}
		if test.ExpectedErr != "" {
			t.Errorf("got nil; want error %q", test.ExpectedErr)
		}
		actualNumManifests := test.State.Manifests.Len()
		if actualNumManifests != test.ExpectedNumManifests {
			t.Errorf("got %d manifests; want %d", actualNumManifests, test.ExpectedNumManifests)
		}
		if (test.DID != sous.DeploymentID{}) {
			m, ok := test.State.Manifests.Get(test.DID.ManifestID)
			if !ok {
				t.Errorf("manifest %q not found", sid.Location)
			}
			_, ok = m.Deployments[test.DID.Cluster]
			if !ok {
				t.Errorf("missing deployment %q", test.DID)
			}
		}
	}
}

func TestUpdateRetryLoop(t *testing.T) {
	dsm := &sous.DummyStateManager{State: sous.NewState()}
	/*
		Source SourceLocation `validate:"nonzero"`
		Flavor string `yaml:",omitempty"`
		Owners []string
		Kind ManifestKind `validate:"nonzero"`
		Deployments DeploySpecs `validate:"keys=nonempty,values=nonzero"`
	*/
	depID := sous.DeploymentID{Cluster: "blah", ManifestID: sous.MustParseManifestID("github.com/user/project")}
	sourceID := sous.MustNewSourceID("github.com/user/project", "", "1.2.3")
	mani := &sous.Manifest{
		Source: sourceID.Location,
		Deployments: sous.DeploySpecs{
			"blah": {Version: semv.MustParse("0.0.0")},
		},
	}
	t.Log(mani.ID())
	dsm.State.Manifests.Add(mani)
	dsm.State.Defs.Clusters = sous.Clusters{"blah": {}}
	user := sous.User{Name: "Judson the Unlucky", Email: "unlucky@opentable.com"}
	deps, err := updateRetryLoop(dsm, sourceID, depID, user)

	assert.NoError(t, err)
	assert.Equal(t, 1, deps.Len())
	dep, present := deps.Get(depID)
	assert.True(t, present)
	assert.Equal(t, "1.2.3", dep.SourceID.Version.String())
	assert.True(t, dsm.ReadCount > 0, "No requests made against state manager")
}

type DummyStateManager struct{}

func (dsm *DummyStateManager) WriteState(s *sous.State, u sous.User) error { return nil }
func (dsm *DummyStateManager) ReadState() (*sous.State, error)             { return nil, nil }

//XXX should actually drive interesting behavior
func TestSousUpdate_Execute(t *testing.T) {
	dsm := &DummyStateManager{}
	su := SousUpdate{
		StateManager:  &graph.StateManager{dsm},
		Manifest:      graph.TargetManifest{Manifest: &sous.Manifest{}},
		ResolveFilter: &graph.RefinedResolveFilter{},
	}
	su.Execute(nil)
}
