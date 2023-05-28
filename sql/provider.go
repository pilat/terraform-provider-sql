package sql

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"dsn": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SQL_DSN", nil),
				Description: "The data source name (DSN) for connecting to the database.",
				Sensitive:   true,
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SQL_TIMEOUT", 600),
				Description: "The maximum amount of time (in seconds) to wait for a connection, zero means wait indefinitely.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"sql": resourceSQL(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (any, error) {
	return &Config{
		DSN:     d.Get("dsn").(string),
		Timeout: d.Get("timeout").(int),
	}, nil
}
