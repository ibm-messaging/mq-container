// +build !linux

package main

// Dummy version of this function, only for non-Linux systems.
// Having this allows unit tests to be run on other platforms (e.g. macOS)
func checkFS(path string) {
	return
}
