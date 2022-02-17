package utils

import (
	"reflect"
	"strings"
	"unicode"
)

var proptypes map[string]string

func addSegment(inrune, segment []rune) []rune {
	if len(segment) == 0 {
		return inrune
	}
	if len(inrune) != 0 {
		inrune = append(inrune, '_')
	}
	inrune = append(inrune, segment...)
	return inrune
}

func camelCaseToUnderscore(str string) string {
	var output []rune
	var segment []rune
	for _, r := range str {

		// not treat number as separate segment
		if !unicode.IsLower(r) && string(r) != "_" && !unicode.IsNumber(r) {
			output = addSegment(output, segment)
			segment = nil
		}
		segment = append(segment, unicode.ToLower(r))
	}
	output = addSegment(output, segment)
	return string(output)
}

// this is used by formatQuery func to cast the different query params to their respective types
func LoadPropTypes(obj interface{}) {
	proptypes = make(map[string]string)
	s := reflect.ValueOf(obj)
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		n := camelCaseToUnderscore(typeOfT.Field(i).Name)
		t := strings.ToLower(f.Type().String())
		switch {
		case strings.Contains(t, "map"):
			proptypes[n] = resolveMapType(t)
		case strings.Contains(t, "int"):
			proptypes[n] = "int"
		case strings.Contains(t, "bool"):
			proptypes[n] = "bool"
		case strings.Contains(t, "float"):
			proptypes[n] = "float"
		}
	}
}

func resolveMapType(t string) string {
	switch {
	case strings.Contains(t, "int"):
		return "int"

	case strings.Contains(t, "bool"):
		return "bool"

	case strings.Contains(t, "float"):
		return "float"
	}
	return "string"
}
