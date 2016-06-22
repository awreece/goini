package goini

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func parseAndFinish(t *testing.T, input string) *RawConfig {
	cp := NewRawConfigParser()
	cp.Parse(strings.NewReader(input))
	ret, err := cp.Finish()
	if err != nil {
		t.Fatalf("Error parsing \n\n%s\ngot %v", input, err)
		return nil
	}
	return ret
}

func checkSection(t *testing.T, name string, actual RawSection,
	expected map[string][]string) {
	for p, vs := range expected {
		if !reflect.DeepEqual(vs, actual.GetPropertyValues(p)) {
			t.Error("Mismatch for section", strconv.Quote(name), "property",
				strconv.Quote(p), ": expected", vs, "but got",
				actual.GetPropertyValues(p))
		}
	}
	for _, p := range actual.Properties() {
		if _, ok := expected[p]; !ok {
			t.Error("Unexpected property", strconv.Quote(p), "in section",
				strconv.Quote(name))
		}
	}
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}

func TestSanity(t *testing.T) {
	c := parseAndFinish(t, "key=value")

	checkSection(t, "global", c.GlobalSection, RawSection{
		"key": {"value"},
	})

	if len(c.Sections()) > 0 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
}

func TestDefaultSection(t *testing.T) {
	c := parseAndFinish(t, joinLines(
		"key=value",
		"key2=",
	))

	checkSection(t, "global", c.GlobalSection, RawSection{
		"key":  {"value"},
		"key2": {""},
	})

	if len(c.Sections()) > 0 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
}

func TestRepeatedKey(t *testing.T) {
	c := parseAndFinish(t, joinLines(
		"key=value",
		"key=value2",
	))

	checkSection(t, "global", c.GlobalSection, RawSection{
		"key": {"value", "value2"},
	})

	if len(c.Sections()) > 0 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
}

func TestComment(t *testing.T) {
	c := parseAndFinish(t, joinLines(
		"key=value # not a comment",
		"; key2=value2",
		"# comment",
		"a=; not a comment",
	))

	checkSection(t, "global", c.GlobalSection, RawSection{
		"key": {"value # not a comment"},
		"a":   {"; not a comment"},
	})

	if len(c.Sections()) > 0 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
}

func TestContinuation(t *testing.T) {
	c := parseAndFinish(t, joinLines(
		"key=value \\",
		"key2=value2",
	))

	checkSection(t, "global", c.GlobalSection, RawSection{
		"key": {"value key2=value2"},
	})

	if len(c.Sections()) > 0 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
}

func TestContinuationIntoEmptyLine(t *testing.T) {
	// We need 2 newlines after the key here to make sure we don't
	// end the string with an empty line (which the Scanner would
	// ignore).
	c := parseAndFinish(t, joinLines(
		"key=\\",
		"",
		"",
	))

	checkSection(t, "global", c.GlobalSection, RawSection{
		"key": {""},
	})

	if len(c.Sections()) > 0 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
}

func TestContinuationIntoComment(t *testing.T) {
	cp := NewRawConfigParser()
	r := strings.NewReader(joinLines(
		"key=\\",
		"; this is a comment",
	))
	if err := cp.Parse(r); err == nil {
		t.Errorf("Expected parse error")
	}
}

func TestInvalidProperty(t *testing.T) {
	var cases = []string{
		"key",
		"[section",
		"[section]a",
		"=value",
	}
	for _, tt := range cases {
		cp := NewRawConfigParser()
		r := strings.NewReader(tt)
		if err := cp.Parse(r); err == nil {
			t.Errorf("Expected parse error for \"%s\"", tt)
		}
	}
}

func TestLeadingWhitespace(t *testing.T) {
	c := parseAndFinish(t, "\n\t\t\t[test1]\n\t\t\tquery=select 1\n\t\t\trate=1")

	checkSection(t, "global", c.GlobalSection, RawSection{})

	if len(c.Sections()) > 1 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
	if section := c.Section("test1"); section == nil {
		t.Errorf("section not found: got %v", c.Sections())
	} else {
		checkSection(t, "test1", section, RawSection{
			"query": []string{"select 1"},
			"rate":  []string{"1"},
		})
	}
}

func TestSectionEmpty(t *testing.T) {
	c := parseAndFinish(t, joinLines(
		"[section]",
	))

	checkSection(t, "global", c.GlobalSection, RawSection{})

	if len(c.Sections()) > 1 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
	if section := c.Section("section"); section == nil {
		t.Errorf("section not found: got %v", c.Sections())
	} else {
		checkSection(t, "section", section, RawSection{})
	}
}

func TestSectionPropery(t *testing.T) {
	c := parseAndFinish(t, joinLines(
		"[section]",
		"key=value",
	))

	checkSection(t, "global", c.GlobalSection, RawSection{})

	if len(c.Sections()) > 1 {
		t.Error("Unexpected sections found: ", c.Sections())
	}
	if section := c.Section("section"); section == nil {
		t.Errorf("section not found: got %v", c.Sections())
	} else {
		checkSection(t, "section", section, RawSection{"key": {"value"}})
	}
}

func TestMultipleSection(t *testing.T) {
	c := parseAndFinish(t, joinLines(
		"[section1]",
		"",
		"[section2]",
	))

	checkSection(t, "global", c.GlobalSection, RawSection{})

	if len(c.Sections()) > 2 {
		t.Error("Unexpected sections found: ", c.Sections())
	}

	if section := c.Section("section1"); section == nil {
		t.Errorf("section not found: got %v", c.Sections())
	} else {
		checkSection(t, "section1", section, RawSection{})
	}

	if section := c.Section("section2"); section == nil {
		t.Errorf("section not found: got %v", c.Sections())
	} else {
		checkSection(t, "section2", section, RawSection{})
	}
}
