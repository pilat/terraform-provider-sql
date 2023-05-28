package sql

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSQL() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSQLCreate,
		ReadContext:   resourceSQLRead,
		UpdateContext: resourceSQLUpdate,
		DeleteContext: resourceSQLDelete,
		SchemaVersion: 1,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta any) error {
			oldVal, newVal := d.GetChange("up")
			oldValStr := oldVal.(string)
			newValStr := newVal.(string)

			// We are not allowing the user to change the `up` attribute after the resource has been created.
			if oldValStr != "" && oldValStr != newValStr {
				return errors.New("changing the `up` attribute is not allowed after the resource has been created")
			}

			return nil
		},
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

func resourceSQLCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	config := meta.(*Config)

	database := d.Get("database").(string)
	up := d.Get("up").(string)

	if err := runSQL(ctx, config, database, up); err != nil {
		return diag.FromErr(err)
	}

	id := fmt.Sprintf("%x", sha256.Sum256([]byte(up)))
	d.SetId(id[:8])

	return nil
}

func resourceSQLRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}

func resourceSQLUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// We are allowing the user to change the `database` and `down` attributes after the resource has been created to
	// give an ability to fix wrong SQL before deleting the resource.

	for _, key := range []string{"database", "down"} {
		if d.HasChange(key) {
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Changing the '%s' will change the terraform state but won't affect the database", key),
			}}
		}
	}

	return nil
}

func resourceSQLDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	config := meta.(*Config)

	database := d.Get("database").(string)
	down := d.Get("down").(string)

	if err := runSQL(ctx, config, database, down); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
