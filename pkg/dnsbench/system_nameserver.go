//go:build !(unix || windows)

package dnsbench

// DefaultNameServer fetches default system name server address.
func DefaultNameServer() string {
	return "127.0.0.1"
}
