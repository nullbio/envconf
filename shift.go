package shift

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

var (
	testHarnessDecodeFile = toml.DecodeFile

	typeTime     = reflect.TypeOf(time.Now())
	typeDuration = reflect.TypeOf(time.Duration(0))

	sizeOfInt = int(reflect.TypeOf(int(0)).Size())
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
	typ := reflect.TypeOf(c)
	if typ.Kind() != reflect.Ptr {
		return errors.Errorf("'c' must be a pointer to a struct, was: %v", typ.String())
	}
	typ = typ.Elem()
	if typ.Kind() != reflect.Struct {
		return errors.Errorf("'c' must be a pointer to a struct, was: %v", typ.String())
	}
	val := reflect.Indirect(reflect.ValueOf(c))

	var i interface{}
	_, err := testHarnessDecodeFile(file, &i)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var m map[string]interface{}
	if i != nil {
		topLevel := i.(map[string]interface{})
		if topLevel != nil {
			envLevel := topLevel[env]
			m = envLevel.(map[string]interface{})
		}
	}

	return bind(typ, val, m)
}

func bind(typ reflect.Type, val reflect.Value, config map[string]interface{}) error {
	n := typ.NumField()
	for i := 0; i < n; i++ {
		f := typ.Field(i)
		key := getKeyName(f)

		if len(key) == 0 {
			continue
		}

		envVal := os.Getenv(strings.ToUpper(key))
		if len(envVal) != 0 {
			fmt.Println("FOUND", key, envVal)
			fieldVal := val.Field(i)

			switch f.Type.Kind() {
			case reflect.String:
				fieldVal.SetString(envVal)
			case reflect.Int:
				i, err := strconv.ParseInt(envVal, 10, sizeOfInt*8)
				if err != nil {
					return err
				}
				fieldVal.SetInt(i)
			case reflect.Int64:
				if f.Type == typeDuration {
					dur, err := time.ParseDuration(envVal)
					if err != nil {
						return err
					}
					fieldVal.SetInt(int64(dur))
				} else {
					i, err := strconv.ParseInt(envVal, 10, 64)
					if err != nil {
						return err
					}
					fieldVal.SetInt(i)
				}
			case reflect.Uint:
				u, err := strconv.ParseUint(envVal, 10, sizeOfInt*8)
				if err != nil {
					return err
				}
				fieldVal.SetUint(u)
			case reflect.Uint64:
				u, err := strconv.ParseUint(envVal, 10, 64)
				if err != nil {
					return err
				}
				fieldVal.SetUint(u)
			case reflect.Float64:
				f, err := strconv.ParseFloat(envVal, 64)
				if err != nil {
					return err
				}
				fieldVal.SetFloat(f)
			case reflect.Struct:
				if f.Type != typeTime {
					return errors.Errorf("failed to bind field %s, unsupported struct type: %s", key, f.Type.String())
				}
				date, err := time.Parse(time.RFC3339, envVal)
				if err != nil {
					return err
				}
				fieldVal.Set(reflect.ValueOf(date))
			default:
				return errors.Errorf("failed to bind field %s, unsupported struct type: %s", key, f.Type.String())
			}
		}

		if intf, ok := config[key]; ok {
			fmt.Println("FOUND", key, intf)
			continue
		}
	}

	return nil
}

func getKeyName(f reflect.StructField) string {
	tag := f.Tag.Get("shift")
	switch {
	case tag == "-":
		return ""
	case len(tag) != 0:
		return tag
	default:
		return toCamel(f.Name)
	}
}

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
