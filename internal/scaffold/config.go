package scaffold

// Config holds the inputs for project scaffolding.
type Config struct {
	ProjectName string // Go module name (e.g. "myapp")
	ProjectPath string // filesystem path (e.g. "./myapp")
	Framework   string
	Database    string
	ORM         string // "none", "gorm", "xorm", "ent"
}
