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
package app

import (
	"fmt"
	migrate "github.com/hchenc/migrator/cmd"

	"github.com/hchenc/migrator/pkg/client"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

var message string

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "generate a new migration file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("----------------")
		fmt.Println("start to generate migration file")
		dataUrl, err := url.Parse(migrate.DatabaseUrl)
		if err != nil {
			panic(err)
		}
		mg := client.NewMigratorClient(dataUrl, migrate.DatabaseUser, migrate.DatabasePass, migrate.MigrationLocation, migrate.MigrationTable, os.Stdout, dump)
		err = mg.New(message)
		if err != nil {
			panic(err)
		}
		fmt.Println("end to generate migration file")
		fmt.Println("----------------")
	},
}

func init() {
	migrate.RootCmd.AddCommand(newCmd)

	newCmd.Flags().StringVarP(&message, "message", "m", "description", "migration file description")

}
