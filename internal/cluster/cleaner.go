package cluster

type Cleaner interface {
	Fetch(clusterName ClusterName) error
	Delete() error
	Print()
}

func CleanupResource(r Cleaner, clusterName ClusterName, dryRun bool) error {
	err := r.Fetch(clusterName)
	if err != nil {
		return err
	}

	r.Print()
	if !dryRun {
		err = r.Delete()
	}

	return err
}
