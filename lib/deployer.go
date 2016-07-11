package sous

type (
	// Deployer describes a complete deployment system, which is able to create,
	// read, update, and delete deployments.
	Deployer interface {
		GetRunningDeployment(fromCluster map[string]string) (Deployments, error)
		RectifyCreates(<-chan *Deployment, chan<- RectificationError)
		RectifyDeletes(<-chan *Deployment, chan<- RectificationError)
		RectifyModifies(<-chan *DeploymentPair, chan<- RectificationError)
	}
)
