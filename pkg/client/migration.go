package client

import (
	"fmt"
	"github.com/hchenc/migrator/pkg/utils"
	"strings"
)

type migrationOptions map[string]string

type MigrationOptions interface {
	Transaction() bool
}

func (m migrationOptions) Transaction() bool {
	return m["transaction"] != "false"
}

type Migration struct {
	Contents string
	Options  MigrationOptions
}

func NewMigration() Migration {
	return Migration{Contents: "", Options: make(migrationOptions)}
}

func parseMigrationContents(contents string) (Migration, Migration, error) {
	up := NewMigration()
	down := NewMigration()

	upDirectiveStart, upDirectiveEnd, hasDefinedUpBlock := utils.GetMatchPositions(contents, utils.UpRegExp)
	downDirectiveStart, downDirectiveEnd, hasDefinedDownBlock := utils.GetMatchPositions(contents, utils.DownRegExp)

	if !hasDefinedUpBlock {
		return up, down, fmt.Errorf("migrator requires each migration to define an up bock with '-- migrate:up'")
	} else if utils.StatementsPrecedeMigrateBlocks(contents, upDirectiveStart, downDirectiveStart) {
		return up, down, fmt.Errorf("migrator does not support statements defined outside of the '-- migrate:up' or '-- migrate:down' blocks")
	}

	upEnd := len(contents)
	downEnd := len(contents)

	if hasDefinedDownBlock && upDirectiveStart < downDirectiveStart {
		upEnd = downDirectiveStart
	} else if hasDefinedDownBlock && upDirectiveStart > downDirectiveStart {
		downEnd = upDirectiveStart
	} else {
		downEnd = -1
	}

	upDirective := utils.Substring(contents, upDirectiveStart, upDirectiveEnd)
	downDirective := utils.Substring(contents, downDirectiveStart, downDirectiveEnd)

	up.Options = parseMigrationOptions(upDirective)
	up.Contents = utils.Substring(contents, upDirectiveStart, upEnd)

	down.Options = parseMigrationOptions(downDirective)
	down.Contents = utils.Substring(contents, downDirectiveStart, downEnd)

	return up, down, nil
}

func parseMigrationOptions(contents string) MigrationOptions {
	options := make(migrationOptions)

	// strip away the -- migrate:[up|down] part
	contents = utils.BlockDirectiveRegExp.ReplaceAllString(contents, "")

	// remove leading and trailing whitespace
	contents = strings.TrimSpace(contents)

	// return empty options if nothing is left to parse
	if contents == "" {
		return options
	}

	// split the options string into pairs, e.g. "transaction:false foo:bar" -> []string{"transaction:false", "foo:bar"}
	stringPairs := utils.WhitespaceRegExp.Split(contents, -1)

	for _, stringPair := range stringPairs {
		// split stringified pair into key and value pairs, e.g. "transaction:false" -> []string{"transaction", "false"}
		pair := utils.OptionSeparatorRegExp.Split(stringPair, -1)

		// if the syntax is well-formed, then store the key and value pair in options
		if len(pair) == 2 {
			options[pair[0]] = pair[1]
		}
	}

	return options
}
