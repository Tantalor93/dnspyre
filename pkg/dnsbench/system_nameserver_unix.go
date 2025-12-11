//go:build unix

package dnsbench

import (
	"bufio"
	"os"
	"strings"
)

const defaultNameServer = "127.0.0.1"

// DefaultNameServer fetches default system name server address based on the /etc/resolv.conf
// If it fails, it returns 127.0.0.1 as default.
func DefaultNameServer() string {
	file, err := open("/etc/resolv.conf")
	if err != nil {
		return defaultNameServer
	}
	defer func() {
		_ = file.Close()
	}()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && (line[0] == ';' || line[0] == '#') {
			// comment line, skip
			continue
		}

		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Split(line, " ")
			if len(fields) == 2 {
				return fields[1]
			}
		}
	}

	return defaultNameServer
}

func open(name string) (*os.File, error) {
	fd, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return fd, nil
}
