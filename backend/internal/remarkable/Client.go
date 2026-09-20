package remarkable

// Config holds the SSH credentials used to reach a tablet. In practice a
// user has one (or a couple) of personal tablets, so — per the setup notes
// in the repo README — these are provided once via env vars rather than
// stored per-Device in the database.
type Config struct {
	User     string
	Password string
	Port     string
}
