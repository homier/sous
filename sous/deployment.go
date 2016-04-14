package sous

type (
	Deployments []Deployment
	// Deployment is a completely configured deployment of a piece of software.
	Deployment struct {
		DeploymentConfig
		// will be given by the execution environment.
		Resources Resources
		// Env is a list of environment variables to set for each instance of
		// of this deployment. It will be checked for conflict with the
		// definitions found in State.Defs.EnvVars, and if not in conflict
		// assumes the greatest priority.
		Env map[string]string
		// NumInstances is a guide to the number of instances that should be
		// deployed in this cluster, note that the actual number may differ due
		// to decisions made by Sous.
		NumInstances int
		// Cluster is the name of the cluster this deployment belongs to. Upon
		// parsing the Manifest, this will be set to the key in
		// Manifests.Deployments which points at this Deployment.
		Cluster string
		// The precise version of the software to be deployed
		NamedVersion
		// Owners is a list of named owners of this repository. The type of this
		// field is subject to change.
		Owners []string
		// Kind is the kind of software that SourceRepo represents.
		Kind ManifestKind
	}

	DeploymentState uint
	LogicalSequence uint

	// Represents deployments commanded by a user
	DeploymentIntentions []DeploymentIntention
	DeploymentIntention  struct {
		Deployment
		// The relative state of this intention
		State DeploymentState
		// The sequence this intention was resolved in - might be e.g. synthesized while walking
		// a git history. This might be left as implicit on the sequence of DIs in a []DI,
		// but if there's a change in storage (i.e. not git), or two single DIs need to be compared,
		// the sequence is useful
		Sequence LogicalSequence
	}
)

const (
	Current    DeploymentState = iota
	Acheived                   = iota
	Waiting                    = iota
	PassedOver                 = iota
)

func BuildDeployment(m *Manifest, spec DeploymentSpec, inherit DeploymentSpecs) (*Deployment, error) {
	return &Deployment{
		Cluster:      spec.clusterName,
		Resources:    spec.Resources,
		Env:          spec.Env,
		NumInstances: spec.NumInstances,
		Owners:       m.Owners,
		Kind:         m.Kind,
		NamedVersion: m.Source.NamedVersion(spec.Version),
	}, nil
}