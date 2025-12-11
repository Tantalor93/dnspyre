//go:build windows

package dnsbench

import (
	"os/exec"
	"regexp"
)

const defaultNameServer = "127.0.0.1"

// DefaultNameServer fetches default system name server address based on the nslookup call.
func DefaultNameServer() string {
	out, err := exec.Command("nslookup").Output()
	if err != nil {
		return defaultNameServer
	}

	// Regex to find DNS Server entries
	re := regexp.MustCompile(`Address:\s+([^\s]+)`)
	matches := re.FindStringSubmatch(string(out))

	if len(matches) != 2 {
		return defaultNameServer
	}
	return matches[1]
}
