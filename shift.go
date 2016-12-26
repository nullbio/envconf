package shift

import (
	"bytes"
	"math"
	"reflect"
	"strconv"
	"time"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

var (
	testHarnessDecodeFile = toml.DecodeFile
)

// Load finds key names from the struct tags in c and tries to load them
// from various sources.
//
// The values that are loaded from the file must be divided in sections for each
// "environment" - that's to say everything must be under top level keys that
// name the environments.
//
// Only a few value types are supported:
// - string
// - int / int64 / uint / uint64
// - time.Time (RFC3339)
// - time.Duration
//
// Earlier sources are overidden by later sources in this list:
// 1. ENV
// 2. File values (top-level keys must be the "env" param to this function)
func Load(c interface{}, file, env string) error {
	var i interface{}

	typ := reflect.TypeOf(c)
	if typ.Kind() != reflect.Ptr {
		return errors.Errorf("'c' must be a pointer to a struct, was: %v", typ.String())
	}
	typ = typ.Elem()
	if typ.Kind() != reflect.Struct {
		return errors.Errorf("'c' must be a pointer to a struct, was: %v", typ.String())
	}

	keys := getKeys(typ)
	_ = keys

	_, err := testHarnessDecodeFile(file, &i)
	if err != nil {
		return err
	}

	return nil
}

func getKeys(t reflect.Type) []string {
	var keys []string

	n := t.NumField()
	for i := 0; i < n; i++ {
		f := t.Field(i)

		if tag := f.Tag.Get("shift"); tag == "-" {
			continue
		} else if len(tag) != 0 {
			keys = append(keys, tag)
		} else {
			keys = append(keys, (f.Name))
		}
	}

	return keys
}

func strToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func strToTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func strToDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

var (
	sizeOfInt = reflect.TypeOf(int(0)).Size()
)

// int64ToInt converts but also checks bounds to ensure it can fit
func int64ToInt(i int64) (int, error) {
	if sizeOfInt == 8 {
		return int(i), nil
	}

	if i > math.MaxInt32 {
		return 0, errors.Errorf("integer too big %d", i)
	} else if i < math.MinInt32 {
		return 0, errors.Errorf("integer too small %d", i)
	}

	return int(i), nil
}

// uint64ToUint converts but also checks bounds to ensure it can fit
func uint64ToUint(u uint64) (uint, error) {
	if sizeOfInt == 8 {
		return uint(u), nil
	}

	if u > math.MaxUint32 {
		return 0, errors.Errorf("unsigned integer too big %d", u)
	}

	return uint(u), nil
}

func toCamel(s string) string {
	b := &bytes.Buffer{}
	var r = []rune(s)

	ln := len(r)
	last, next := false, false
	for i := 0; i < ln; i++ {
		upper := unicode.IsUpper(r[i])

		if i != 0 && upper {
			next = i+1 < ln && unicode.IsUpper(r[i+1])

			if !last || !next {
				b.WriteByte('_')
			}
		}

		b.WriteRune(unicode.ToLower(r[i]))
		last = upper
	}

	return b.String()
}
