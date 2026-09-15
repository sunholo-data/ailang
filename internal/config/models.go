package config

// The model registry (internal/modelreg): which models.yml is authoritative.
const (
	EnvModelsPath         = "AILANG_MODELS_PATH"
	EnvModelsPublishedDir = "AILANG_MODELS_PUBLISHED_DIR"
)

// DefaultModelsPublishedDir is where the published registry is mounted
// (gcsfuse) in the fleet containers.
const DefaultModelsPublishedDir = "/registry"

var modelVars = []Var{
	{EnvModelsPath, "", AreaModels, "Explicit models.yml to load; a file named here that fails to parse is a hard error, and `models publish` publishes it (default internal/modelreg/models.yml)."},
	{EnvModelsPublishedDir, DefaultModelsPublishedDir, AreaModels, "Directory of the published registry (the gcsfuse mount); a broken file there degrades to the embedded floor, loudly."},
}

// ModelsPath returns AILANG_MODELS_PATH, "" when unset.
func ModelsPath() string { return get(EnvModelsPath) }

// ModelsPublishedDir returns AILANG_MODELS_PUBLISHED_DIR, default /registry.
func ModelsPublishedDir() string { return getOr(EnvModelsPublishedDir) }
