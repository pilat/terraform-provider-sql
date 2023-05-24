package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/pilat/terraform-provider-sql/sql"
)

func main() {
	var debugMode bool = true
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{
		ProviderFunc: sql.Provider,
	}

	if debugMode {
		err := plugin.Debug(context.Background(), "pilat/sql", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}
