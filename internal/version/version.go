package version

import "fmt"

var (
	BuildVersion string
	BuildDate    string
	BuildCommit  string
)

// PrintBuildInfo prints the build version, date, and commit to standard output.
func PrintBuildInfo() {
	v := BuildVersion
	if v == "" {
		v = "N/A"
	}
	d := BuildDate
	if d == "" {
		d = "N/A"
	}
	c := BuildCommit
	if c == "" {
		c = "N/A"
	}
	fmt.Printf("Build version: %s\n", v)
	fmt.Printf("Build date: %s\n", d)
	fmt.Printf("Build commit: %s\n", c)
}
