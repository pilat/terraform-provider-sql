package sql

import (
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSQL() *schema.Resource {
	return &schema.Resource{
		Create:        resourceSQLCreate,
		Read:          resourceSQLRead,
		Update:        resourceSQLUpdate,
		Delete:        resourceSQLDelete,
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"database": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The database to run the migration against.",
			},
			"up": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The SQL command to run when migrating up.",
			},
			"down": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The SQL command to run when migrating down.",
			},
		},
	}
}

func resourceSQLCreate(d *schema.ResourceData, meta any) error {
	config := meta.(*Config)

	database := d.Get("database").(string)
	up := d.Get("up").(string)

	err := runSQL(config, database, up)
	if err != nil {
		return err
	}

	id := fmt.Sprintf("sql-%x", sha256.Sum256([]byte(up)))
	d.SetId(id)

	return resourceSQLRead(d, meta)
}

func resourceSQLRead(d *schema.ResourceData, meta any) error {
	return nil
}

func resourceSQLUpdate(d *schema.ResourceData, meta any) error {
	if d.HasChange("database") || d.HasChange("up") {
		return fmt.Errorf("changing any attribute of an existing SQL resource is not allowed")
	}
	return nil
}

func resourceSQLDelete(d *schema.ResourceData, meta any) error {
	config := meta.(*Config)

	database := d.Get("database").(string)
	down := d.Get("down").(string)

	err := runSQL(config, database, down)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
