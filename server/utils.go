package server

import (
	"fmt"
	"regexp"
	"time"

	"github.com/thoas/go-funk"
)

var (
	re = regexp.MustCompile(`https:\/\/(?:telegram|t)\.me\/joinchat\/([A-Za-z0-9-]{22})`)
)

func getHashes(in string) []string {
	hashes := make([]string, 0)
	for _, match := range re.FindAllStringSubmatch(in, -1) {
		hashes = append(hashes, match[1])
	}

	return funk.UniqString(hashes)
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02d:%02d", h, m)
}
