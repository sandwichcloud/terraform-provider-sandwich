package sandwich

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func getProject(d *schema.ResourceData, config *Config) (string, error) {
	return getProjectFromSchema("project", d, config)
}

func getProjectFromSchema(projectSchemaField string, d *schema.ResourceData, config *Config) (string, error) {
	res, ok := d.GetOk(projectSchemaField)
	if ok && projectSchemaField != "" {
		return res.(string), nil
	}
	if config.ProjectName != "" {
		return config.ProjectName, nil
	}
	return "", fmt.Errorf("%s: required field is not set", projectSchemaField)
}
