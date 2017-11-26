package argparse

import (
	"fmt"
	"os"
	"strings"
)

type arg struct {
	result   interface{} // Pointer to the resulting value
	opts     *Options    // Options
	sname    string      // Short name (in parser will start with "-"
	lname    string      // Long name (in parser will start with "--"
	size     int         // Size defines how many args after match will need to be consumed
	unique   bool        // Specifies whether flag should be present only ones
	parsed   bool        // Specifies whether flag has been parsed already
	fileFlag int         // File mode to open file with
	filePerm os.FileMode // File permissions to set a file
	selector *[]string   // Used in Selector type to allow to choose only one from list of options
	parent   *command    // Used to get access to specific command
}

type help struct{}

func (o *arg) check(argument string) bool {
	// Shortcut to showing help
	if argument == "-h" || argument == "--help" {
		helpText := o.parent.Usage()
		fmt.Print(helpText)
		os.Exit(0)
	}

	// Check for long name only if not empty
	if o.lname != "" {
		// If argument begins with "--" and next is not "-" then it is a long name
		if len(argument) > 2 && strings.HasPrefix(argument, "--") && argument[2] != '-' {
			if argument[2:] == o.lname {
				return true
			}
		}
	}
	// Check for short name only if not empty
	if o.sname != "" {
		// If argument begins with "-" and next is not "-" then it is a short name
		if len(argument) > 1 && strings.HasPrefix(argument, "-") && argument[1] != '-' {
			switch o.result.(type) {
			case *bool:
				// For flags we allow multiple shorthand in one
				if strings.Contains(argument[1:], o.sname) {
					return true
				}
			default:
				// For all other types it must be separate argument
				if argument[1:] == o.sname {
					return true
				}
			}
		}
	}

	return false
}

func (o *arg) reduce(position int, args *[]string) {
	argument := (*args)[position]
	// Check for long name only if not empty
	if o.lname != "" {
		// If argument begins with "--" and next is not "-" then it is a long name
		if len(argument) > 2 && strings.HasPrefix(argument, "--") && argument[2] != '-' {
			if argument[2:] == o.lname {
				for i := position; i < position+o.size; i++ {
					(*args)[i] = ""
				}
			}
		}
	}
	// Check for short name only if not empty
	if o.sname != "" {
		// If argument begins with "-" and next is not "-" then it is a short name
		if len(argument) > 1 && strings.HasPrefix(argument, "-") && argument[1] != '-' {
			switch o.result.(type) {
			case *bool:
				// For flags we allow multiple shorthand in one
				if strings.Contains(argument[1:], o.sname) {
					(*args)[position] = strings.Replace(argument, o.sname, "", -1)
					if (*args)[position] == "-" {
						(*args)[position] = ""
					}
				}
			default:
				// For all other types it must be separate argument
				if argument[1:] == o.sname {
					for i := position; i < position+o.size; i++ {
						(*args)[i] = ""
					}
				}
			}
		}
	}
}

func (o *arg) parse(args []string) error {
	// If unique do not allow more than one time
	if o.unique && o.parsed {
		return fmt.Errorf("[%s] can only be present once", o.name())
	}

	// If validation function provided -- execute, on error return it immediately
	if o.opts != nil && o.opts.Validate != nil {
		err := o.opts.Validate(args)
		if err != nil {
			return err
		}
	}

	switch o.result.(type) {
	case *help:
		helpText := o.parent.Usage()
		fmt.Print(helpText)
		os.Exit(0)
	case *bool:
		*o.result.(*bool) = true
		o.parsed = true
	case *string:
		if len(args) < 1 {
			return fmt.Errorf("[%s] must be followed by a string", o.name())
		}
		if len(args) > 1 {
			return fmt.Errorf("[%s] followed by too many arguments", o.name())
		}
		// Selector case
		if o.selector != nil {
			match := false
			for _, v := range *o.selector {
				if args[0] == v {
					match = true
				}
			}
			if !match {
				return fmt.Errorf("bad value for [%s]. Allowed values are %v", o.name(), *o.selector)
			}
		}
		*o.result.(*string) = args[0]
		o.parsed = true
	case *os.File:
		if len(args) < 1 {
			return fmt.Errorf("[%s] must be followed by a path to file", o.name())
		}
		if len(args) > 1 {
			return fmt.Errorf("[%s] followed by too many arguments", o.name())
		}
		f, err := os.OpenFile(args[0], o.fileFlag, o.filePerm)
		if err != nil {
			return err
		}
		*o.result.(*os.File) = *f
		o.parsed = true
	case *[]string:
		if len(args) < 1 {
			return fmt.Errorf("[%s] must be followed by a string", o.name())
		}
		if len(args) > 1 {
			return fmt.Errorf("[%s] followed by too many arguments", o.name())
		}
		*o.result.(*[]string) = append(*o.result.(*[]string), args[0])
		o.parsed = true
	default:
		return fmt.Errorf("unsupported type [%t]", o.result)
	}
	return nil
}

func (o *arg) name() string {
	var name string
	if o.lname == "" {
		name = "-" + o.sname
	} else if o.sname == "" {
		name = "--" + o.lname
	} else {
		name = "-" + o.sname + "|" + "--" + o.lname
	}
	return name
}

func (o *arg) usage() string {
	var result string
	result = o.name()
	switch o.result.(type) {
	case *bool:
		break
	case *string:
		if o.selector != nil {
			result = result + " (" + strings.Join(*o.selector, "|") + ")"
		} else {
			result = result + " \"<value>\""
		}
	case *os.File:
		result = result + " <file>"
	case *[]string:
		result = result + " \"<string>\""
	default:
		break
	}
	if o.opts == nil || o.opts.Required == false {
		result = "[" + result + "]"
	}
	return result
}
