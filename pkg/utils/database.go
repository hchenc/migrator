package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func GetSchameName(u *url.URL) string {
	name := u.Path
	if len(name) > 0 && name[:1] == "/" {
		name = name[1:]
	}
	return name
}

func FormateDatabaseStr(str string) string {
	str = strings.Replace(str, "`", "\\`", -1)
	return fmt.Sprintf("`%s`", str)
}
