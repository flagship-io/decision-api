package utils

import (
	"regexp"
	"strconv"
)

var mapExp = regexp.MustCompile(`^(.+)\[(.+)]$`)

// formatQuery returns a casted map string interface of a query string
func formatQuery(query map[string]string) map[string]interface{} {
	parentmap := make(map[string]interface{})
	for k, e := range query {
		value := e
		if keyIsMap(k) {
			buildChildMap(k, value, parentmap)
			continue
		}
		rv, err := castValue(k, value)
		if err == nil {
			parentmap[k] = rv
		}

	}
	return parentmap
}

func castValue(k string, value string) (interface{}, error) {
	if t, ok := proptypes[k]; ok {
		switch t {
		case "int":
			return strconv.Atoi(value)
		case "bool":
			return strconv.ParseBool(value)
		case "float":
			return strconv.ParseFloat(value, 32)

		}
	}
	return value, nil
}

func buildChildMap(k string, value interface{}, parentmap map[string]interface{}) {
	pk, ck := explodeMapKey(k)
	childmap, ok := parentmap[pk]
	if !ok {
		childmap = make(map[string]interface{})
	}
	if v, ok := childmap.(map[string]interface{}); ok {
		rv, err := castValue(pk, value.(string))
		if err == nil {
			v[ck] = rv
			parentmap[pk] = v
		}

	}
}
func explodeMapKey(k string) (parentKey string, childKey string) {
	matches := mapExp.FindStringSubmatch(k)
	return matches[1], matches[2]
}
func keyIsMap(k string) bool {
	return mapExp.MatchString(k)
}
