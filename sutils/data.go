package sutils

import (
	"bytes"
	"encoding/binary"
	"reflect"
)

// ConvertToBytes converts any struct to a byte slice
func ConvertToBytes(data any, bigEndians ...bool) ([]byte, error) {
	var buf bytes.Buffer
	bigEndian := false
	if len(bigEndians) > 0 {
		bigEndian = bigEndians[0]
	}
	err := binaryWrite(&buf, data, bigEndian)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// binaryWrite writes the binary representation of data to the buffer
func binaryWrite(buf *bytes.Buffer, data any, bigEndian bool) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var order binary.ByteOrder
	order = binary.LittleEndian
	if bigEndian {
		order = binary.BigEndian
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if err := binaryWrite(buf, field.Interface(), bigEndian); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := binaryWrite(buf, v.Index(i).Interface(), bigEndian); err != nil {
				return err
			}
		}
	case reflect.String:
		if err := binary.Write(buf, order, []byte(v.String())); err != nil {
			return err
		}
	default:
		if err := binary.Write(buf, order, data); err != nil {
			return err
		}
	}
	return nil
}
