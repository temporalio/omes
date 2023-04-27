package omes

import (
	"fmt"
	"reflect"
)

// OptionsToFlags constructs command line flags from an options struct.
// Struct fields must strings annotated with a "flag" tag.
func OptionsToFlags(opt interface{}) (flags []string) {
	elem := reflect.TypeOf(opt).Elem()
	val := reflect.ValueOf(opt)
	numFields := elem.NumField()

	for idx := 0; idx < numFields; idx++ {
		field := elem.Field(idx)
		fieldVal := reflect.Indirect(val).FieldByName(field.Name).String()
		if fieldVal != "" {
			flags = append(flags, fmt.Sprintf("--%s", field.Tag.Get("flag")), fieldVal)
		}
	}
	return
}

// OptionToFlagName gets the "flag" struct tag of an options struct field.
func OptionToFlagName(options interface{}, fieldName string) string {
	field, ok := reflect.TypeOf(options).Elem().FieldByName(fieldName)
	if !ok {
		panic(fmt.Errorf("options struct %v does not have a field named %s", options, fieldName))
	}
	return field.Tag.Get("flag")
}
