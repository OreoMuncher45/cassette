package buildinfo

import (
	"fmt"
	"io"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func Text() string {
	return fmt.Sprintf(
		"version=%s\ncommit=%s\nbuild_date=%s\n",
		Version,
		Commit,
		BuildDate,
	)
}

func PrintVersion(w io.Writer) error {
	_, err := io.WriteString(w, Text())
	return err
}
