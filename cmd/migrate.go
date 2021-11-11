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
package cmd

import (
	"fmt"
	"github.com/hchenc/migrator/pkg/migrator"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

var dump bool

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("----------------")
		fmt.Println("start to migrate")
		dataUrl, err := url.Parse(databaseUrl)
		if err != nil {
			panic(err)
		}
		mg := migrator.NewMigrator(dataUrl, databaseUser, databasePass, migrationLocation, migrationTable, os.Stdout, dump)
		err = mg.Migrate()
		if err != nil {
			panic(err)
		}
		fmt.Println("end to migrate")
		fmt.Println("----------------")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// migrateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	migrateCmd.Flags().BoolVar(&dump, "d", false, "auto dump schema before migrate")
}