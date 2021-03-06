/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package options

import (
	"fmt"
	migrate "github.com/hchenc/migrator/cmd"
	"github.com/hchenc/migrator/pkg/client"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

var dump bool

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("----------------")
		fmt.Println("start to migrate")
		dataUrl, err := url.Parse(migrate.DatabaseUrl)
		if err != nil {
			panic(err)
		}
		mg := client.NewMigratorClient(dataUrl, migrate.DatabaseUser, migrate.DatabasePass, migrate.MigrationLocation, migrate.MigrationTable, os.Stdout, dump)
		err = mg.Migrate()
		if err != nil {
			panic(err)
		}
		fmt.Println("end to migrate")
		fmt.Println("----------------")
	},
}

func init() {
	migrate.RootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolVarP(&dump, "autodump", "a", false, "auto dump schema before migrate")
}
