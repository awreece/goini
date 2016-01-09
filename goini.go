// A config parser for .ini files, written in pure go.
//
// In this dialect:
//
//   - Comments are lines starting with either a '#' or a ';'.
//   - A line ending with a '\' continues onto the next line.
//   - It is illegal to have a continuation before a comment or the end of
//     file.
//
package goini

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Warning: Prefer to use the public methods since the type of RawSection
// might change.
type RawSection map[string][]string

type RawConfig struct {
	GlobalSection RawSection
	sections      map[string]RawSection
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
	if vs, ok := s[property]; !ok {
		return "", false
	} else {
		return json.Number(strings.Join(vs, " ")), true
	}
}

// Returns the list of unique properties that have been set at least once.
func (s RawSection) Properties() []string {
	keys := []string{}
	for p, _ := range s {
		keys = append(keys, p)
	}
	return keys
}

// Returns the list of unique sections in the config object.
func (c *RawConfig) Sections() map[string]RawSection {
	return c.sections
}

func NewRawConfigParser() *RawConfigParser {
	config := &RawConfig{make(RawSection), make(map[string]RawSection)}
	return &RawConfigParser{config, config.GlobalSection, "", nil}
}

func (cp *RawConfigParser) parseLine(line string) error {
	if cp.err != nil {
		return cp.err
	}

	if len(line) > 0 && (line[0] == ';' || line[0] == '#') {
		if cp.currentLine != "" {
			cp.err = errors.New("Invalid continuation into comment line.")
			return cp.err
		} else {
			return nil
		}
	}

	if len(line) > 0 && line[len(line)-1] == '\\' {
		cp.currentLine += line[:len(line)-1]
		return nil
	} else {
		line = cp.currentLine + line
		cp.currentLine = ""
	}

	if len(strings.TrimSpace(line)) == 0 {
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
		cp.err = errors.New("Invalid section header start character")
		return cp.err
	}

	parts := strings.SplitN(line[1:], "]", 2)
	if len(parts) != 2 {
		cp.err = errors.New("No section header end character found")
		return cp.err
	}
	if parts[1] != "" {
		cp.err = errors.New("Trailing characters after section header")
		return cp.err
	}

	return cp.addSection(parts[0])
}

func (cp *RawConfigParser) parseProperty(line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		cp.err = errors.New("Invalid property line")
		return cp.err
	}

	cp.currentSection.addProperty(parts[0], parts[1])
	return nil
}

// Returns the new config object or the first error encountered while parsing.
//
// Also resets the config parser.
func (cp *RawConfigParser) Finish() (*RawConfig, error) {
	retConfig, retError := cp.config, cp.err
	cp.config = &RawConfig{make(RawSection), make(map[string]RawSection)}
	cp.currentSection = cp.config.GlobalSection
	cp.err = nil
	if retError != nil {
		return nil, retError
	} else {
		return retConfig, nil
	}
}

func (cp *RawConfigParser) addSection(name string) error {
	if _, ok := cp.config.sections[name]; ok {
		cp.err = errors.New(fmt.Sprint("Duplicate section name", strconv.Quote(name)))
		return cp.err
	}

	cp.currentSection = make(map[string][]string)
	cp.config.sections[name] = cp.currentSection
	return nil
}

func (cp *RawConfigParser) Parse(file io.Reader) error {
	if cp.err != nil {
		return cp.err
	}

	line := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line += 1

		if err := cp.parseLine(scanner.Text()); err != nil {
			return errors.New(fmt.Sprintf(
				"error parsing line %d %v", line, err))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if cp.currentLine != "" {
		return errors.New(fmt.Sprintf(
			"error parsing line %d: continuation at end of file", line))
	}
	return nil
}

func (cp *RawConfigParser) ParseFile(filename string) error {
	if cp.err != nil {
		return cp.err
	}

	if file, err := os.Open(filename); err != nil {
		return errors.New(fmt.Sprintf("error opening file %s: %v",
			strconv.Quote(filename), err))
	} else {
		return cp.Parse(file)
	}
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
