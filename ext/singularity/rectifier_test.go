package singularity

import (
	"log"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* TESTS BEGIN */

func TestBuildDeployRequest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	rID := "expectedRID"
	dr, err := buildDeployRequest(sous.Deployable{
		BuildArtifact: &sous.BuildArtifact{
			Name: "an-image",
			Type: "docker",
		},
		Deployment: &sous.Deployment{
			SourceID: sous.SourceID{
				Location: sous.SourceLocation{
					Repo: "repo",
				},
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
				Resources:    sous.Resources{},
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "http://cluster",
			},
		},
	}, rID, map[string]string{})
	require.NoError(err)
	assert.NotNil(dr)
	assert.Equal(dr.Deploy.RequestId, rID)
}

func TestDockerMetadataSet(t *testing.T) {
	logTempl := "expected:%s got:%s"
	testKey := "expectedKey"
	testValue := "expectedValue"
	md := map[string]string{
		testKey: testValue,
	}

	rID := "expectedRID"
	dr, err := buildDeployRequest(sous.Deployable{
		BuildArtifact: &sous.BuildArtifact{
			Name: "an-image",
			Type: "docker",
		},
		Deployment: &sous.Deployment{
			SourceID: sous.SourceID{
				Location: sous.SourceLocation{
					Repo: "repo",
				},
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
				Resources:    sous.Resources{},
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "http://cluster",
			},
		},
	}, rID, md)

	if err != nil {
		t.Fatal(err)
	}
	if dr.Deploy.Metadata[testKey] == testValue {
		t.Logf(logTempl, testValue, dr.Deploy.Metadata[testKey])
	} else {
		t.Fatalf(logTempl, testValue, dr.Deploy.Metadata[testKey])
	}
}

func baseDeployablePair() *sous.DeployablePair {
	return &sous.DeployablePair{
		ExecutorData: &singularityTaskData{requestID: "reqid"},
		Prior: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				Name: "the-prior-image",
				Type: "docker",
			},
			Deployment: &sous.Deployment{
				SourceID: sous.SourceID{
					Location: sous.SourceLocation{
						Repo: "fake.tld/org/project",
					},
				},
				DeployConfig: sous.DeployConfig{
					NumInstances: 1,
					Resources:    sous.Resources{},
				},
				ClusterName: "cluster",
				Cluster: &sous.Cluster{
					BaseURL: "cluster",
				},
			},
		},
		Post: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				Name: "the-post-image",
				Type: "docker",
			},
			Deployment: &sous.Deployment{
				SourceID: sous.SourceID{
					Location: sous.SourceLocation{
						Repo: "fake.tld/org/project",
					},
				},
				DeployConfig: sous.DeployConfig{
					NumInstances: 1,
					Resources:    sous.Resources{},
				},
				ClusterName: "cluster",
				Cluster: &sous.Cluster{
					BaseURL: "cluster",
				},
			},
		},
	}

}

func TestModifyScale(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	assert := assert.New(t)
	mods := make(chan *sous.DeployablePair, 1)
	errs := make(chan sous.DiffResolution, 10)

	pair := baseDeployablePair()
	pair.Prior.Deployment.DeployConfig.NumInstances = 12
	pair.Post.Deployment.DeployConfig.NumInstances = 24

	client := sous.NewDummyRectificationClient()

	deployer := NewDeployer(client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		if e.Error != nil {
			t.Error(e)
		}
	}

	assert.Len(client.Deployed, 0)
	if assert.Len(client.Created, 1) {
		assert.Equal(24, client.Created[0].Deployment.DeployConfig.NumInstances)
	}
}

func TestModifyImage(t *testing.T) {
	assert := assert.New(t)

	before := "1.2.3-test"
	after := "2.3.4-new"
	pair := baseDeployablePair()
	pair.Prior.Deployment.SourceID.Version = semv.MustParse(before)
	pair.Post.Deployment.SourceID.Version = semv.MustParse(after)
	pair.Post.BuildArtifact.Name = "2.3.4"

	mods := make(chan *sous.DeployablePair, 1)
	log := make(chan sous.DiffResolution, 10)

	client := sous.NewDummyRectificationClient()
	deployer := NewDeployer(client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, log)
	close(log)

	for e := range log {
		if e.Error != nil {
			t.Error(e.Error)
		}
	}

	assert.Len(client.Created, 0)

	if assert.Len(client.Deployed, 1) {
		assert.Regexp("2.3.4", client.Deployed[0].BuildArtifact.Name)
	}
}

func TestModifyResources(t *testing.T) {
	assert := assert.New(t)
	version := "1.2.3-test"

	pair := baseDeployablePair()

	pair.Prior.Deployment.SourceID.Version = semv.MustParse(version)
	pair.Prior.Deployment.Resources["memory"] = "100"

	pair.Post.Deployment.SourceID.Version = semv.MustParse(version)
	pair.Post.Deployment.Resources["memory"] = "500"
	pair.Post.BuildArtifact.Name = "1.2.3"

	mods := make(chan *sous.DeployablePair, 1)
	log := make(chan sous.DiffResolution, 10)

	client := sous.NewDummyRectificationClient()
	deployer := NewDeployer(client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, log)
	close(log)

	for e := range log {
		if e.Error != nil {
			t.Error(e)
		}
	}

	assert.Len(client.Created, 0)

	if assert.Len(client.Deployed, 1) {
		assert.Regexp("1.2.3", client.Deployed[0].BuildArtifact.Name)
		assert.Regexp("500", client.Deployed[0].Deployment.DeployConfig.Resources["memory"])
	}
}

func TestModify(t *testing.T) {
	assert := assert.New(t)
	before := "1.2.3-test"
	after := "2.3.4-new"

	pair := baseDeployablePair()

	pair.Prior.Deployment.SourceID.Version = semv.MustParse(before)
	pair.Prior.Deployment.DeployConfig.NumInstances = 1
	pair.Prior.Deployment.DeployConfig.Volumes = sous.Volumes{{"host", "container", "RO"}}

	pair.Post.Deployment.SourceID.Version = semv.MustParse(after)
	pair.Post.Deployment.DeployConfig.NumInstances = 24
	pair.Post.Deployment.DeployConfig.Volumes = sous.Volumes{{"host", "container", "RW"}}
	pair.Post.BuildArtifact.Name = "2.3.4"

	mods := make(chan *sous.DeployablePair, 1)
	results := make(chan sous.DiffResolution, 10)

	client := sous.NewDummyRectificationClient()
	deployer := NewDeployer(client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, results)
	close(results)

	for e := range results {
		if e.Error != nil {
			t.Error(e)
		}
	}

	if assert.Len(client.Created, 1) {
		assert.Equal(24, client.Created[0].Deployment.DeployConfig.NumInstances)
	}

	if assert.Len(client.Deployed, 1) {
		assert.Regexp("2.3.4", client.Deployed[0].BuildArtifact.Name)
		t.Logf("VOLUMES:%#v", client.Deployed[0].Deployment.DeployConfig.Volumes)
		assert.Equal("RW", string(client.Deployed[0].Deployment.DeployConfig.Volumes[0].Mode))
	}

}

func TestDeletes(t *testing.T) {
	assert := assert.New(t)

	deleted := &sous.DeployablePair{
		ExecutorData: &singularityTaskData{requestID: "reqid"},
		Prior: &sous.Deployable{
			Deployment: &sous.Deployment{
				SourceID: sous.SourceID{
					Location: sous.SourceLocation{
						Repo: "fake.tld/org/project",
					},
				},
				DeployConfig: sous.DeployConfig{
					NumInstances: 12,
				},
				ClusterName: "",
				Cluster: &sous.Cluster{
					BaseURL: "cluster",
				},
			},
		},
	}

	dels := make(chan *sous.DeployablePair, 1)
	log := make(chan sous.DiffResolution, 10)

	client := sous.NewDummyRectificationClient()
	deployer := NewDeployer(client)

	dels <- deleted
	close(dels)
	deployer.RectifyDeletes(dels, log)
	close(log)

	for e := range log {
		if e.Error != nil {
			t.Error(e)
		}
	}

	assert.Len(client.Deployed, 0)
	assert.Len(client.Created, 0)

	// We no longer expect any deletions; See deployer.RectifySingleDelete.
	//expectedDeletions := 1
	expectedDeletions := 0

	assert.Len(client.Deleted, expectedDeletions)
	//if assert.Len(client.Deleted, expectedDeletions) {
	// We no longer expect any deletions; See deployer.RectifySingleDelete.
	//req := client.Deleted[0]
	//assert.Equal("cluster", req.Cluster)
	//assert.Equal("reqid::", req.Reqid)
	//}
}

func TestCreates(t *testing.T) {
	assert := assert.New(t)

	created := &sous.DeployablePair{
		Post: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				Type: "docker",
				Name: "reqid,0.0.0",
			},
			Deployment: &sous.Deployment{
				SourceID: sous.SourceID{
					Location: sous.SourceLocation{
						Repo: "fake.tld/org/project",
					},
				},
				DeployConfig: sous.DeployConfig{
					NumInstances: 12,
				},
				Cluster:     &sous.Cluster{BaseURL: "cluster"},
				ClusterName: "nick",
			},
		},
	}

	crts := make(chan *sous.DeployablePair, 1)
	log := make(chan sous.DiffResolution, 10)

	client := sous.NewDummyRectificationClient()
	deployer := NewDeployer(client)

	crts <- created
	close(crts)
	deployer.RectifyCreates(crts, log)
	close(log)

	for e := range log {
		if e.Error != nil {
			t.Error(e)
		}
	}

	if assert.Len(client.Deployed, 1) {
		dep := client.Deployed[0]
		assert.Equal("cluster", dep.Deployment.Cluster.BaseURL)
		assert.Equal("reqid,0.0.0", dep.BuildArtifact.Name)
	}

	if assert.Len(client.Created, 1) {
		req := client.Created[0]
		assert.Equal("cluster", req.Deployment.Cluster.BaseURL)
		reqID, err := computeRequestID(&req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Regexp("^project---nick-[0-9a-f]*$", reqID)
		assert.Equal(12, req.Deployment.DeployConfig.NumInstances)
	}
}
