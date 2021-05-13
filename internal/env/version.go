package env

import (
	"os"
	"os/exec"
	"strings"
)

func GetBuildVersion() (versionInfo VersionInfo, err error) {
	if BuildVersion == "" {
		os.Setenv("WEKACTL_FORCE_DEV", "1")
		var out []byte
		out, err = exec.Command("./scripts/get_build_params.sh").Output()
		if err != nil {
			return
		}
		result := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
		BuildVersion = result[0]
		Commit = result[1]
	}

	versionInfo.BuildVersion = BuildVersion
	versionInfo.Commit = Commit
	return
}
