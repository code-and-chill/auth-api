package filter

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// Filter is a struct for filtering data.
type Filter map[string]string

// GetInt returns query params by key.
func (f Filter) GetInt(key string) interface{} {
	mapVal := f[key]

	if mapVal == "" {
		return nil
	}
	val, err := strconv.Atoi(mapVal)
	if err != nil {
		return nil
	}
	return val
}

// GetFloat returns query params by key.
func (f Filter) GetFloat(key string) interface{} {
	mapVal := f[key]

	if mapVal == "" {
		return nil
	}
	val, err := strconv.ParseFloat(mapVal, 64)
	if err != nil {
		return nil
	}
	return val
}

// GetString returns query params by key.
func (f Filter) GetString(key string) interface{} {
	return f[key]
}

// GetBool returns query params by key.
func (f Filter) GetBool(key string) interface{} {
	mapVal := f[key]

	if mapVal == "" {
		return nil
	}
	val, err := strconv.ParseBool(mapVal)
	if err != nil {
		return nil
	}
	return val
}

// GetTime returns query params by key.
func (f Filter) GetTime(key string) interface{} {
	mapVal := f[key]

	if mapVal == "" {
		return nil
	}
	val, err := time.Parse("2006-01-02 15:04:05", key)
	if err != nil {
		return nil
	}
	return val
}

// ParseLimitOffset parses limit and offset.
func ParseLimitOffset(strLimit, strOffset string) (ok bool, limit, offset int) {
	var err error
	limit, err = strconv.Atoi(strLimit)
	if err != nil {
		return false, 0, 0
	}
	offset, err = strconv.Atoi(strOffset)
	if err != nil {
		return false, 0, 0
	}
	return true, limit, offset
}

// HashCode generates hashcode for this filter.
func (f Filter) HashCode() (string, error) {
	jsonByte, err := json.Marshal(f)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return fmt.Sprintf("%x", md5.Sum(jsonByte)), nil
}
