package utils

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var MigrationFileRegexp = regexp.MustCompile(`^\d.*\.sql$`)
var UpRegExp = regexp.MustCompile(`(?m)^--\s*migrate:up(\s*$|\s+\S+)`)
var DownRegExp = regexp.MustCompile(`(?m)^--\s*migrate:down(\s*$|\s+\S+)$`)
var EmptyLineRegExp = regexp.MustCompile(`^\s*$`)
var CommentLineRegExp = regexp.MustCompile(`^\s*--`)
var WhitespaceRegExp = regexp.MustCompile(`\s+`)
var OptionSeparatorRegExp = regexp.MustCompile(`:`)
var BlockDirectiveRegExp = regexp.MustCompile(`^--\s*migrate:[up|down]]`)

func MustFindMigrationFiles(dir string, re *regexp.Regexp) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("could not find migrations directory `%s`", dir))
	}

	matches := []string{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !re.MatchString(name) {
			continue
		}

		matches = append(matches, name)
	}

	sort.Strings(matches)
	if len(matches) == 0 {
		panic(fmt.Sprintf("no migration files found"))
	}

	return matches
}

func MustFindMigrationFile(dir string, ver string) string {
	if ver == "" {
		panic("migration version is required")
	}

	ver = regexp.QuoteMeta(ver)
	re := regexp.MustCompile(fmt.Sprintf(`^%s.*\.sql$`, ver))

	files := MustFindMigrationFiles(dir, re)

	return files[0]
}

func MigrationVersion(filename string) string {
	return regexp.MustCompile(`^\d+`).FindString(filename)
}

func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("unable to create directory `%s`", dir)
	}

	return nil
}

func StatementsPrecedeMigrateBlocks(contents string, upDirectiveStart, downDirectiveStart int) bool {
	until := upDirectiveStart

	if downDirectiveStart > -1 {
		until = Min(upDirectiveStart, downDirectiveStart)
	}

	lines := strings.Split(contents[0:until], "\n")

	for _, line := range lines {
		if IsEmptyLine(line) || IsCommentLine(line) {
			continue
		}
		return true
	}

	return false
}

// isEmptyLine will return true if the line has no
// characters or if all the characters are whitespace characters
func IsEmptyLine(s string) bool {
	return EmptyLineRegExp.MatchString(s)
}

// isCommentLine will return true if the line is a SQL comment
func IsCommentLine(s string) bool {
	return CommentLineRegExp.MatchString(s)
}

func GetMatchPositions(s string, re *regexp.Regexp) (int, int, bool) {
	match := re.FindStringIndex(s)
	if match == nil {
		return -1, -1, false
	}
	return match[0], match[1], true
}

func Substring(s string, begin, end int) string {
	if begin == -1 || end == -1 {
		return ""
	}
	return s[begin:end]
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
