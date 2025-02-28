package sutils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
)

// ConvertToBytes converts any struct to a byte slice
func ConvertToBytes(data any, bigEndian bool) ([]byte, error) {
	var buf bytes.Buffer
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
	if v.Kind() != reflect.Struct {
		return errors.New("data is not a struct or pointer to struct")
	}

	var order binary.ByteOrder
	order = binary.LittleEndian
	if bigEndian {
		order = binary.BigEndian
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() == reflect.Struct {
			if err := binaryWrite(buf, field.Addr().Interface(), bigEndian); err != nil {
				return err
			}
		} else {
			if err := binary.Write(buf, order, field.Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}
