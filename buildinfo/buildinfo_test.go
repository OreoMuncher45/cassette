package buildinfo

import "testing"

func TestText(t *testing.T) {
	prevVersion := Version
	prevCommit := Commit
	prevBuildDate := BuildDate
	t.Cleanup(func() {
		Version = prevVersion
		Commit = prevCommit
		BuildDate = prevBuildDate
	})

	Version = "1.2.3"
	Commit = "abc123"
	BuildDate = "2026-04-13T00:00:00Z"

	got := Text()
	want := "version=1.2.3\ncommit=abc123\nbuild_date=2026-04-13T00:00:00Z\n"
	if got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}
