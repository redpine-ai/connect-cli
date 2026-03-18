package version

var (
	Version = "dev"
	Commit  = "none"
)

func Full() string {
	return "connect " + Version + " (" + Commit + ")"
}
