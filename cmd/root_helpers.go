package cmd

import "os"

// projectRoot returns the directory that tssk uses as its working root.
// By default this is the current working directory.  Set the TSSK_ROOT
// environment variable to override (useful for testing).
func projectRoot() string {
	if r := os.Getenv("TSSK_ROOT"); r != "" {
		return r
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
