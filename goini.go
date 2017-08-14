// Package goini provides a config parser for .ini files.
//
// In this dialect:
//
//   - Comments are lines starting with either a '#' or a ';'.
//   - A line ending with a '\' continues onto the next line.
//   - It is illegal to have a continuation before a comment or the end of
//     file.
//   - All leading and trailing whitespace is stripped from properties
//     and values.
//
package goini

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type DecodeOptionKind int

const (
	UniqueOption DecodeOptionKind = iota
	MultiOption                   = iota
)

type DecodeOption struct {
	Kind  DecodeOptionKind
	Usage string
	Parse func(string, interface{}) error
}

type DecodeOptionSet map[string]*DecodeOption

// Warning: Prefer to use the public methods since the type of RawSection
// might change.
type RawSection map[string][]string

type RawConfig struct {
	GlobalSection RawSection
	sections      map[string]RawSection

	// Ordered based on order in the config file.
	sectionNames []string
}

// An object for parsing config files and building a RawConfig. Can be
// used to parse and merge multiple config files. Uses the "Errors are values"
// pattern -- intermediate steps can return error, but only the final error
// returned from Finish() needs to be checked.
type RawConfigParser struct {
	config         *RawConfig
	currentSection RawSection
	currentLine    string
	err            error
	numLines       int
}

func (s RawSection) addProperty(property, value string) {
	s[property] = append(s[property], value)
}

// Returns all the values set for a property or the empty list nil if has
// never been set.
func (s RawSection) GetPropertyValues(property string) []string {
	return s[property]
}

// If the property has been set at least once, returns all values joined
// as a space separated string. Returns true if the propery has been set
// at least once.
func (s RawSection) GetPropertyNumber(property string) (json.Number, bool) {
	vs, ok := s[property]
	if !ok {
		return "", false
	}
	return json.Number(strings.Join(vs, " ")), true
}

// Returns the list of unique properties that have been set at least once.
func (s RawSection) Properties() []string {
	keys := []string{}
	for p := range s {
		keys = append(keys, p)
	}
	return keys
}

func (dos DecodeOptionSet) Decode(section RawSection, dest interface{}) error {
	for _, property := range section.Properties() {
		option, ok := dos[property]
		if !ok {
			return fmt.Errorf("unexpected property %s",
				strconv.Quote(property))
		}
		values := section.GetPropertyValues(property)
		if option.Kind == UniqueOption && len(values) != 1 {
			return fmt.Errorf("property %s cannot be repeated",
				strconv.Quote(property))
		}
		for _, value := range values {
			if e := option.Parse(value, dest); e != nil {
				return fmt.Errorf("error parsing %s: %s",
					strconv.Quote(property), e)
			}
		}
	}
	return nil
}

// Return the section by name or nil if the section
// was not defined.
//
func (c *RawConfig) Section(n string) RawSection {
	return c.sections[n]
}

// Returns the list of unique sections in the config object.
func (c *RawConfig) Sections() []string {
	return c.sectionNames
}

func NewRawConfigParser() *RawConfigParser {
	config := &RawConfig{make(RawSection), make(map[string]RawSection), nil}
	return &RawConfigParser{config, config.GlobalSection, "", nil, 0}
}

func (cp *RawConfigParser) parseLine(line string) error {
	if cp.err != nil {
		return cp.err
	}

	if len(line) > 0 && (line[0] == ';' || line[0] == '#') {
		if cp.currentLine != "" {
			cp.err = fmt.Errorf("Invalid continuation into comment line on line %d.", cp.numLines)
			return cp.err
		}
		return nil
	}

	if len(line) > 0 && line[len(line)-1] == '\\' {
		cp.currentLine += line[:len(line)-1]
		return nil
	}
	line = strings.TrimSpace(cp.currentLine + line)
	cp.currentLine = ""

	if len(line) == 0 {
		return nil
	}

	if line[0] == '[' {
		if cp.err = cp.parseSectionHeader(line); cp.err != nil {
			return cp.err
		}
	} else if cp.err = cp.parseProperty(line); cp.err != nil {
		return cp.err
	}

	return nil
}

func (cp *RawConfigParser) parseSectionHeader(line string) error {
	if line[0] != '[' {
		cp.err = fmt.Errorf("Invalid section header start character on line %d", cp.numLines)
		return cp.err
	}

	parts := strings.SplitN(line[1:], "]", 2)
	if len(parts) != 2 {
		cp.err = fmt.Errorf("No section header end character found on line %d", cp.numLines)
		return cp.err
	}
	if parts[1] != "" {
		cp.err = fmt.Errorf("Trailing characters after section header on line %d", cp.numLines)
		return cp.err
	}

	return cp.addSection(parts[0])
}

func (cp *RawConfigParser) parseProperty(line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		cp.err = fmt.Errorf("Invalid property on line %d", cp.numLines)
		return cp.err
	}

	p := strings.TrimSpace(parts[0])
	v := strings.TrimSpace(parts[1])
	cp.currentSection.addProperty(p, v)
	return nil
}

// Returns the new config object or the first error encountered while parsing.
//
// Also resets the config parser.
func (cp *RawConfigParser) Finish() (*RawConfig, error) {
	retConfig, retError := cp.config, cp.err
	*cp = *NewRawConfigParser()
	if retError != nil {
		return nil, retError
	}
	return retConfig, nil
}

func (cp *RawConfigParser) addSection(name string) error {
	if _, ok := cp.config.sections[name]; ok {
		cp.err = fmt.Errorf("Duplicate section name %s on line %d", strconv.Quote(name), cp.numLines)
		return cp.err
	}

	cp.currentSection = make(map[string][]string)
	cp.config.sections[name] = cp.currentSection
	cp.config.sectionNames = append(cp.config.sectionNames, name)

	return nil
}

func (cp *RawConfigParser) Parse(file io.Reader) error {
	if cp.err != nil {
		return cp.err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cp.numLines++

		if err := cp.parseLine(scanner.Text()); err != nil {
			return fmt.Errorf("error parsing line %d %v",
				cp.numLines, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if cp.currentLine != "" {
		return fmt.Errorf(
			"error parsing line %d: continuation at end of file", cp.numLines)
	}
	return nil
}

func (cp *RawConfigParser) ParseFile(filename string) error {
	if cp.err != nil {
		return cp.err
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	return cp.Parse(file)
}

func ParseFile(filename string) (*RawConfig, error) {
	cp := NewRawConfigParser()
	if err := cp.ParseFile(filename); err != nil {
		return nil, err
	}
	return cp.Finish()
}

func Parse(reader io.Reader) (*RawConfig, error) {
	cp := NewRawConfigParser()
	if err := cp.Parse(reader); err != nil {
		return nil, err
	}
	return cp.Finish()
}
