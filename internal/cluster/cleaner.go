package cluster

type Cleaner interface {
	Fetch() error
	Delete() error
	Print()
}

func CleanupResource(r Cleaner, dryRun bool) error {
	err := r.Fetch()
	if err != nil {
		return err
	}

	r.Print()
	if !dryRun {
		err = r.Delete()
	}

	return err
}
