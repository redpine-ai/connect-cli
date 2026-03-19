package version

var (
	Version = "dev"
	Commit  = "none"
)

func Full() string {
	return "redpine " + Version + " (" + Commit + ")"
}
