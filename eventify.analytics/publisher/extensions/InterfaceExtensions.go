package extensions

import "reflect"

func GetType(i interface{}) string {
	if i == nil {
		return "<nil>"
	}

	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.String()
}
