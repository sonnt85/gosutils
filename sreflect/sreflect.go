package sreflect

import (
	"fmt"
	"reflect"
	"unicode"
)

func StructUniqueByFieldName(structSlice interface{}, fieldName string) (retlist []interface{}) {
	if len(fieldName) != 0 && unicode.IsLower([]rune(fieldName)[0]) {
		return
	}
	keys := make(map[interface{}]bool)
	uniFunc := func(s reflect.Value) {
		for i := 0; i < s.Len(); i++ {
			entry := s.Index(i).Interface()
			v := reflect.ValueOf(entry)
			if v.Kind() != reflect.Struct {
				continue
			}

			vField := v.FieldByName(fieldName)
			if !vField.IsValid() {
				continue
			}

			keyInterface := vField.Interface()
			if _, ok := keys[keyInterface]; !ok {
				keys[keyInterface] = true
				retlist = append(retlist, entry)
			}
		}
	}
	switch reflect.TypeOf(structSlice).Kind() {
	case reflect.Slice, reflect.Array:
		s := reflect.ValueOf(structSlice)
		uniFunc(s)
	case reflect.Ptr:
		e := reflect.ValueOf(structSlice).Elem()
		switch e.Kind() {
		case reflect.Slice, reflect.Array:
			uniFunc(e)
			//			pt := reflect.PtrTo(reflect.ValueOf(structSlice).Type()) // create a *structSlice type.
			//			pv := reflect.New(pt.Elem())                             // create a reflect.Value of type *T.
			//			v := reflect.ValueOf(retlist)
			//			pv.Elem().Set(v) // sets pv to point to underlying value of v.
		}
	}
	return
}

// Reflect if an interface is either a struct or a pointer to a struct
// and has the defined member method. If error is nil, it means
// the MethodName is accessible with reflect.
func ReflectStructMethod(Iface interface{}, MethodName string) error {
	ValueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// Get the method by name
	Method := ValueIface.MethodByName(MethodName)
	if !Method.IsValid() {
		return fmt.Errorf("Couldn't find method `%s` in interface `%s`, is it Exported?", MethodName, ValueIface.Type())
	}
	return nil
}

func SlideHasElem(s interface{}, elem interface{}) bool {
	arrV := reflect.ValueOf(s)

	if arrV.Kind() == reflect.Slice || arrV.Kind() == reflect.Array {
		for i := 0; i < arrV.Len(); i++ {

			// XXX - panics if slice element points to an unexported struct field
			// see https://golang.org/pkg/reflect/#Value.Interface
			if arrV.Index(i).Interface() == elem {
				return true
			}
		}
	}

	return false
}

// Reflect if an interface is either a struct or a pointer to a struct
// and has the defined member field, if error is nil, the given
// FieldName exists and is accessible with reflect.
func ReflectStructField(Iface interface{}, FieldName string) error {
	ValueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface's Type, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// 'dereference' with Elem() and get the field by name
	Field := ValueIface.Elem().FieldByName(FieldName)
	if !Field.IsValid() {
		return fmt.Errorf("interface `%s` does not have the field `%s`", ValueIface.Type(), FieldName)
	}
	return nil
}
