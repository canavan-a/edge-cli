package cmd

// repo and version are baked in at build time via -ldflags.
// The sentinel values below are replaced during a real build;
// commands that need them will error if the sentinels are still present.
var (
	repo    = "REPO_NOT_SET"
	version = "dev"
)

func isDevBuild() bool {
	return repo == "REPO_NOT_SET"
}
