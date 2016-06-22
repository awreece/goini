package goini

import (
	"fmt"
	"strconv"
	"strings"
)

func Example_section() {
	cp := NewRawConfigParser()
	cp.Parse(strings.NewReader(`
[section]
message=hello
	`))
	if config, err := cp.Finish(); err == nil {
		section := config.Section("section")
		message := strings.Join(section.GetPropertyValues("message"), " ")
		fmt.Println(message)
	}
	// Output: hello
}

func Example_globalSection() {
	cp := NewRawConfigParser()
	cp.Parse(strings.NewReader(`
message=hello
message=world
	`))
	if config, err := cp.Finish(); err == nil {
		message := strings.Join(config.GlobalSection.GetPropertyValues("message"), " ")
		fmt.Println(message)
	}
	// Output: hello world
}

func Example_continuation() {
	cp := NewRawConfigParser()
	cp.Parse(strings.NewReader(`
message=hello \
world
	`))
	if config, err := cp.Finish(); err == nil {
		message := strings.Join(config.GlobalSection.GetPropertyValues("message"), " ")
		fmt.Println(message)
	}
	// Output: hello world
}

func Example_comment() {
	cp := NewRawConfigParser()
	cp.Parse(strings.NewReader(`
message=hello
; message=world
	`))
	if config, err := cp.Finish(); err == nil {
		messageParts := config.GlobalSection.GetPropertyValues("message")
		fmt.Println(strings.Join(messageParts, " "))
	}
	// Output: hello
}

func ExampleRawSection_GetPropertyNumber() {
	cp := NewRawConfigParser()
	cp.Parse(strings.NewReader(`
number=1
message=hello
message=world
	`))
	if config, err := cp.Finish(); err == nil {
		number, _ := config.GlobalSection.GetPropertyNumber("number")
		if i, e := number.Int64(); e == nil {
			fmt.Println(i)
		}
		message, _ := config.GlobalSection.GetPropertyNumber("message")
		fmt.Println(message)
	}
	// Output: 1
	// hello world
}

func ExampleDecodeOptionSet_Decode() {
	rawSection := RawSection{
		"key": []string{"1"},
	}

	spec := DecodeOptionSet{
		"key": &DecodeOption{Kind: UniqueOption,
			Usage: "A help message.",
			Parse: func(v string, d interface{}) error {
				i, e := strconv.Atoi(v)
				if e != nil {
					return e
				}
				*d.(*int) = i
				return nil
			},
		},
	}

	var key int
	spec.Decode(rawSection, &key)
	fmt.Println(key)

	//Output: 1
}
