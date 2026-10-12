package gemini

import (
	"context"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
)

// WithPreparedConfiguration freezes configured Vertex project/location
// selection without metadata, gcloud or credential discovery. If no project
// was configured, default metadata/gcloud discovery remains deferred to setup.
func WithPreparedConfiguration() ClientOption {
	resolve := config.PrepareCloudProjectResolver()
	location := strings.TrimSpace(config.GoogleCloudLocation())
	return func(c *Client) { c.preparedProject = resolve; c.preparedLocation = true; c.location = location }
}

func (c *Client) resolveProject() (string, error) {
	if c.preparedProject == nil {
		return getGCPProject()
	}
	if project, err := c.preparedProject(context.Background()); err == nil {
		return project, nil
	}
	return discoverGcloudProject()
}
