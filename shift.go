// Package shift enables loading of configuration via environment and files
// but makes a distinction between environments in the file portion. As a Go
// package it also does the convenient thing of putting this configuration
// into a struct whose names are configurable via struct tags.
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

	typeTime      = reflect.TypeOf(time.Now())
	typeDuration  = reflect.TypeOf(time.Duration(0))
	typeStringArr = reflect.TypeOf([]string{})

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
// - bool
// - string / []string
// - int / int64 / uint / uint64
// - time.Time (RFC3339)
// - time.Duration
//
// Earlier sources are overidden by later sources in this list:
// 1. ENV
// 2. File values (top-level keys must be the "env" param to this function)
func Load(c interface{}, file, envPrefix, env string) error {
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

	return bind(envPrefix, typ, val, m)
}

func bind(envPrefix string, typ reflect.Type, val reflect.Value, config map[string]interface{}) error {
	n := typ.NumField()
	for i := 0; i < n; i++ {
		f := typ.Field(i)
		key := getKeyName(f)

		if len(key) == 0 {
			continue
		}

		envKey := key
		if len(envPrefix) != 0 {
			envKey = fmt.Sprintf("%s_%s", envPrefix, envKey)
		}
		envVal := os.Getenv(strings.ToUpper(envKey))
		if len(envVal) != 0 {
			if err := assignFromEnv(envVal, f.Type, val.Field(i)); err != nil {
				return errors.Wrapf(err, "failed to assign key %s", key)
			}
			continue
		}

		if intf, ok := config[key]; ok {
			if err := assignFromIntf(intf, f.Type, val.Field(i)); err != nil {
				return errors.Wrapf(err, "failed to assign key %s", key)
			}
			continue
		}
	}

	return nil
}

func assignFromEnv(envVal string, fieldType reflect.Type, fieldVal reflect.Value) error {
	switch fieldType.Kind() {
	case reflect.String:
		fieldVal.SetString(envVal)
	case reflect.Bool:
		if envVal != "true" && envVal != "false" {
			return errors.Errorf("invalid value for bool, must be %q or %q: %s", "true", "false", envVal)
		}
		if envVal == "true" {
			fieldVal.SetBool(true)
		} else {
			fieldVal.SetBool(false)
		}
	case reflect.Int:
		i, err := strconv.ParseInt(envVal, 10, sizeOfInt*8)
		if err != nil {
			return err
		}
		fieldVal.SetInt(i)
	case reflect.Int64:
		if fieldType == typeDuration {
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
		if fieldType != typeTime {
			return errors.Errorf("unsupported struct type: %s", fieldType.String())
		}
		date, err := time.Parse(time.RFC3339, envVal)
		if err != nil {
			return err
		}
		fieldVal.Set(reflect.ValueOf(date))
	default:
		return errors.Errorf("unsupported struct type: %s", fieldType.String())
	}

	return nil
}

func assignFromIntf(val interface{}, fieldType reflect.Type, fieldVal reflect.Value) error {
	switch fieldType.Kind() {
	case reflect.String:
		if s, ok := val.(string); ok {
			fieldVal.SetString(s)
			return nil
		}
	case reflect.Bool:
		if b, ok := val.(bool); ok {
			fieldVal.SetBool(b)
			return nil
		}
	case reflect.Int:
		if i, ok := val.(int64); ok {
			if _, err := int64ToInt(i); err != nil {
				return err
			}
			fieldVal.SetInt(i)
			return nil
		}
	case reflect.Int64:
		if fieldType == typeDuration {
			if s, ok := val.(string); ok {
				d, err := time.ParseDuration(s)
				if err != nil {
					return err
				}
				fieldVal.Set(reflect.ValueOf(d))
				return nil
			}
		} else {
			if i, ok := val.(int64); ok {
				if _, err := int64ToInt(i); err != nil {
					return err
				}
				fieldVal.SetInt(i)
				return nil
			}
		}
	case reflect.Uint:
		if i, ok := val.(int64); ok {
			if _, err := uint64ToUint(uint64(i)); err != nil {
				return err
			}
			fieldVal.SetUint(uint64(i))
			return nil
		}
	case reflect.Uint64:
		if i, ok := val.(int64); ok {
			fieldVal.SetUint(uint64(i))
			return nil
		}
	case reflect.Float64:
		if f, ok := val.(float64); ok {
			fieldVal.SetFloat(f)
			return nil
		}
	case reflect.Struct:
		if fieldType == typeTime {
			fieldVal.Set(reflect.ValueOf(val))
			return nil
		}
	case reflect.Slice:
		if fieldType == typeStringArr {
			if s, ok := val.([]interface{}); ok {
				sArr := make([]string, len(s))
				for i := range s {
					str, _ := s[i].(string)
					sArr[i] = str
				}
				fieldVal.Set(reflect.ValueOf(sArr))
				return nil
			}
		}
	}

	return errors.Errorf("unsupported conversion %s -> %s", fieldType.String(), reflect.TypeOf(val).String())
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
	var last bool
	for i := 0; i < ln; i++ {
		upper := unicode.IsUpper(r[i])

		if i != 0 && upper {
			if !last {
				b.WriteByte('_')
			} else if i+1 < ln && !unicode.IsUpper(r[i+1]) {
				b.WriteByte('_')
			}
		}

		b.WriteRune(unicode.ToLower(r[i]))
		last = upper
	}

	return b.String()
}
