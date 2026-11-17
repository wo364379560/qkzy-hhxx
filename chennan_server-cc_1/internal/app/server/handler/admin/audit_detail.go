package handler_admin

import (
	"strconv"
	"strings"
)

func summarizeUintIDs(ids []uint) (int, string) {
	const maxShown = 10

	if len(ids) == 0 {
		return 0, ""
	}

	seen := make(map[uint]struct{}, len(ids))
	unique := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	shownCount := len(unique)
	if shownCount > maxShown {
		shownCount = maxShown
	}

	values := make([]string, 0, shownCount)
	for _, id := range unique[:shownCount] {
		values = append(values, strconv.FormatUint(uint64(id), 10))
	}

	summary := strings.Join(values, ",")
	if len(unique) > maxShown {
		summary += "...(total=" + strconv.Itoa(len(unique)) + ")"
	}

	return len(unique), summary
}
