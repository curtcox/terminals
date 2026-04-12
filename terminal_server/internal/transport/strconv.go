package transport

import "strconv"

func toString(v int64) string {
	return strconv.FormatInt(v, 10)
}
