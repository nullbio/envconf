package shift

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

var testToml = `
[prod]
configstring   = "badstring"
[test]
configstring   = "badstring"
[dev]
configstring   = "string"
configint      = -5
configint64    = -10
configuint     = 5
configuint64   = 10
configfloat64  = 10.5
configtime     = 2006-01-02T15:04:05Z
configduration = "15s"
`

type testStruct struct {
	Envstring   string
	Envint      int
	Envint64    int64
	Envuint     uint
	Envuint64   uint64
	Envfloat64  float64
	Envtime     time.Time
	Envduration time.Duration

	Configstring   string
	Configint      int
	Configint64    int64
	Configuint     uint
	Configuint64   uint64
	Configfloat64  float64
	Configtime     time.Time
	Configduration time.Duration
}

func TestLoad(t *testing.T) {
	restore := testHarnessDecodeFile
	os.Setenv("ENVSTRING", "string")
	os.Setenv("ENVINT", "-5")
	os.Setenv("ENVINT64", "-10")
	os.Setenv("ENVUINT", "5")
	os.Setenv("ENVUINT64", "10")
	os.Setenv("ENVFLOAT64", "10.5")
	os.Setenv("ENVTIME", "2006-01-02T15:04:05Z")
	os.Setenv("ENVDURATION", "15s")
	defer func() {
		testHarnessDecodeFile = restore
		os.Setenv("ENVSTRING", "")
		os.Setenv("ENVINT", "")
		os.Setenv("ENVINT64", "")
		os.Setenv("ENVUINT", "")
		os.Setenv("ENVUINT64", "")
		os.Setenv("ENVFLOAT64", "")
		os.Setenv("ENVTIME", "")
		os.Setenv("ENVDURATION", "")
	}()

	testHarnessDecodeFile = func(_ string, i interface{}) (toml.MetaData, error) {
		return toml.Decode(testToml, i)
	}

	got := testStruct{}
	want := testStruct{
		Envstring:   "string",
		Envint:      -5,
		Envint64:    -10,
		Envuint:     5,
		Envuint64:   10,
		Envfloat64:  10.5,
		Envtime:     time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		Envduration: 15 * time.Second,

		Configstring:   "string",
		Configint:      -5,
		Configint64:    -10,
		Configuint:     5,
		Configuint64:   10,
		Configfloat64:  10.5,
		Configtime:     time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		Configduration: 15 * time.Second,
	}

	err := Load(&got, "file", "", "dev")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("didn't load keys properly:\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestLoadEnvPrefix(t *testing.T) {
	restore := testHarnessDecodeFile
	os.Setenv("PREFIX_ENVSTRING", "string")
	defer func() {
		testHarnessDecodeFile = restore
		os.Setenv("PREFIX_ENVSTRING", "")
	}()

	testHarnessDecodeFile = func(_ string, i interface{}) (toml.MetaData, error) {
		return toml.Decode(testToml, i)
	}

	got := testStruct{}
	want := testStruct{
		Envstring: "string",
	}

	err := Load(&got, "file", "prefix", "dev")
	if err != nil {
		t.Fatal(err)
	}

	if got.Envstring != want.Envstring {
		t.Error("Envstring was wrong:", got.Envstring)
	}
}

func TestGetKeyName(t *testing.T) {
	t.Parallel()

	var s = struct {
		Int       int
		IDMonster int
		String    string `shift:"a"`
		Uint      uint   `shift:"-"`
	}{}

	typ := reflect.TypeOf(s)

	if getKeyName(typ.Field(0)) != "int" {
		t.Error("int wasn't found")
	}
	if getKeyName(typ.Field(1)) != "id_monster" {
		t.Error("id_monster wasn't found")
	}
	if getKeyName(typ.Field(2)) != "a" {
		t.Error("a wasn't found")
	}
	if getKeyName(typ.Field(3)) != "" {
		t.Error("uint should be empty string")
	}
}

func TestInt64ToInt(t *testing.T) {
	t.Parallel()

	i, err := int64ToInt(int64(23))
	if err != nil {
		t.Error(err)
	}

	if i != 23 {
		t.Error("i wrong:", i)
	}
}

func TestUint64ToUint(t *testing.T) {
	t.Parallel()

	u, err := uint64ToUint(uint64(23))
	if err != nil {
		t.Error(err)
	}

	if u != 23 {
		t.Error("u wrong:", u)
	}
}

func TestToCamel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		In  string
		Out string
	}{
		{"oneTwo", "one_two"},
		{"OneTwo", "one_two"},
		{"OneTWOThree", "one_two_three"},
		{"ONETWOThree", "onetwo_three"},
		{"TestONE", "test_one"},
		{"TestOneTWO", "test_one_two"},
		{"TestO", "test_o"},
		{"TestON", "test_on"},
		{"TestOn", "test_on"},
		{"TEST", "test"},
	}

	for i, test := range tests {
		if got := toCamel(test.In); got != test.Out {
			t.Errorf("%d) [%s] %s != %s", i, test.In, got, test.Out)
		}
	}
}

var testTomlTwo = `
[dev]
	bind = ":3999"
	log_json = true
	assets_no_hash = true
[prod]
`

type testStructTwo struct {
	// env vars
	TLSBind          string `toml:"tls_bind"`
	AssetsNoCompress bool   `toml:"assets_no_compress"`
	LogJSONTwo       bool   `toml:"log_json_two"`
	// config vars
	Bind         string `toml:"bind"`
	LogJSON      bool   `toml:"log_json"`
	AssetsNoHash bool   `toml:"assets_no_hash"`
	// ignore vars
	TestVAR bool `toml:"test_var"`
}

func TestLoadTwo(t *testing.T) {
	restore := testHarnessDecodeFile
	os.Setenv("TLS_BIND", ":4000")
	os.Setenv("ASSETS_NO_COMPRESS", "true")
	os.Setenv("LOG_JSON_TWO", "true")
	defer func() {
		testHarnessDecodeFile = restore
		os.Setenv("TLS_BIND", "")
		os.Setenv("ASSETS_NO_COMPRESS", "")
		os.Setenv("LOG_JSON_TWO", "")
	}()

	testHarnessDecodeFile = func(_ string, i interface{}) (toml.MetaData, error) {
		return toml.Decode(testTomlTwo, i)
	}

	got := testStructTwo{TestVAR: true}
	want := testStructTwo{
		TLSBind:          ":4000",
		AssetsNoCompress: true,
		LogJSONTwo:       true,

		Bind:         ":3999",
		LogJSON:      true,
		AssetsNoHash: true,

		TestVAR: true, // Ensure this is still true & Load didn't overwrite to zero val
	}

	err := Load(&got, "file", "", "dev")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("didn't load keys properly:\nwant: %#v\ngot: %#v", want, got)
	}
}
