//go:build !unix

// default implementation for non-unix systems
package dnsbench

// DefaultNameServer fetches default system name server address.
func DefaultSystemNameServerAddress() string {
	return "127.0.0.1"
}
