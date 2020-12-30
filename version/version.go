package version

import "fmt"

const version string = "v0.1.10"

// GetHumanVersion return version
func GetHumanVersion() string {
	return fmt.Sprintf("tsf-go %s", version)
}
