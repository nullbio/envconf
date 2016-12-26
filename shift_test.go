package shift

import (
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

func TestLoad(t *testing.T) {
	restore := testHarnessDecodeFile
	defer func() {
		testHarnessDecodeFile = restore
	}()

	testHarnessDecodeFile = func(_ string, i interface{}) (toml.MetaData, error) {
		return toml.MetaData{}, errors.New("hello")
	}

	var str = struct {
		Hello string `shift:"안녕"`
		World string
	}{}

	err := Load(&str, "file", "dev")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetKeys(t *testing.T) {
	t.Parallel()

	var s = struct {
		Int    int
		String string `shift:"a"`
		Uint   uint   `shift:"-"`
	}{}

	keys := getKeys(reflect.TypeOf(s))

	sort.Strings(keys)
	if len(keys) != 2 {
		t.Fatalf("wanted 2 keys, got %d: %#v", len(keys), keys)
	}
	if keys[0] != "a" {
		t.Error("a wasn't found")
	}
	if keys[1] != "int" {
		t.Error("int wasn't found")
	}
}

func TestStrToInt(t *testing.T) {
	t.Parallel()

	i, err := strToInt("558")
	if err != nil {
		t.Error(err)
	}

	if i != 558 {
		t.Error("i is wrong", i)
	}
}

func TestStrToTime(t *testing.T) {
	t.Parallel()

	ti, err := strToTime("2006-01-02T15:04:05Z")
	if err != nil {
		t.Error(err)
	}

	want := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	if !want.Equal(ti) {
		t.Error("ti wrong:", ti)
	}
}

func TestStrToDuration(t *testing.T) {
	t.Parallel()

	dur, err := strToDuration("46s")
	if err != nil {
		t.Error(err)
	}

	if dur != time.Second*46 {
		t.Error("dur wrong:", dur)
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
	}

	for i, test := range tests {
		if got := toCamel(test.In); got != test.Out {
			t.Errorf("%d) [%s] %s != %s", i, test.In, got, test.Out)
		}
	}
}
