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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var cfgFile string

var MigrationLocation string
var MigrationTable string
var DatabaseUrl string
var DatabaseUser string
var DatabasePass string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "migrator",
	Short: "A lightweight database migration tool",
	Long: `A lightweight database migration tool which 
provide command cli and rest api to manage the database.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	cobra.CheckErr(RootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./src/main/resources/application.properties", "migrator config file")

	RootCmd.PersistentFlags().StringVarP(&MigrationLocation, "migration-location", "d", "./db/migration", "migration file directory where to store migration script")
	RootCmd.PersistentFlags().StringVarP(&MigrationTable, "migration-table", "t", "schema_history", "database table name where to store schema change record")
	RootCmd.PersistentFlags().StringVarP(&DatabaseUrl, "database-url", "l", "", "database url")
	RootCmd.PersistentFlags().StringVarP(&DatabaseUser, "database-user", "u", "", "database user")
	RootCmd.PersistentFlags().StringVarP(&DatabasePass, "database-password", "p", "", "database password")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	viper.SetConfigFile(cfgFile)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Fprintln(os.Stderr, "Config file not found: ", viper.ConfigFileUsed())
		} else {
			fmt.Fprintln(os.Stderr, "Error while loading config file: ", viper.ConfigFileUsed(), err.Error())
		}
	} else {
		for flag, value := range map[*string]string{
			&MigrationLocation: "migrator.location",
			&MigrationTable:    "migrator.table",
			&DatabaseUrl:       "spring.datasource.url",
			&DatabaseUser:      "spring.datasource.username",
			&DatabasePass:      "spring.datasource.password",
		} {
			if viper.GetString(value) != "" {
				*flag = viper.GetString(value)
			}
		}
		DatabaseUrl = strings.Split(DatabaseUrl[5:], "?")[0]
	}
}
