package slogrus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sort"

	"github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/ppjson"
)

type fieldKey string

// FieldMap allows customization of the key names for default fields.
type FieldMap map[fieldKey]string

func (f FieldMap) resolve(key fieldKey) string {
	if k, ok := f[key]; ok {
		return k
	}

	return string(key)
}

// JSONFormatter formats logs into parsable json
type JSONFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	// The format to use is the same than for time.Format or time.Parse from the standard
	// library.
	// The standard Library already provides a set of predefined format.
	TimestampFormat string

	// DisableTimestamp allows disabling automatic timestamps in output
	DisableTimestamp bool

	// DisableHTMLEscape allows disabling html escaping in output
	DisableHTMLEscape bool

	// DataKey allows users to put all the log entry parameters into a nested dictionary at a given key.
	DataKey string

	// FieldMap allows users to customize the names of keys for default fields.
	// As an example:
	// formatter := &JSONFormatter{
	//   	FieldMap: FieldMap{
	// 		 FieldKeyTime:  "@timestamp",
	// 		 FieldKeyLevel: "@level",
	// 		 FieldKeyMsg:   "@message",
	// 		 FieldKeyFunc:  "@caller",
	//    },
	// }
	FieldMap FieldMap

	// CallerPrettyfier can be set by the user to modify the content
	// of the function and file keys in the json data when ReportCaller is
	// activated. If any of the returned value is the empty string the
	// corresponding key will be removed from json fields.
	CallerPrettyfier func(*runtime.Frame) (function string, file string)

	// PrettyPrint will indent all json logs
	PrettyPrint          bool
	ReorderArrayKeys     []string
	DisableMsgJsonOpject bool
}

// This is to not silently overwrite `time`, `msg`, `func` and `level` fields when
// dumping it. If this code wasn't there doing:
//
//	logrus.WithField("level", 1).Info("hello")
//
// Would just silently drop the user provided level. Instead with this code
// it'll logged as:
//
//	{"level": "info", "fields.level": 1, "msg": "hello", "time": "..."}
//
// It's not exported because it's still using Data in an opinionated way. It's to
// avoid code duplication between the two default formatters.
func prefixFieldClashes(data logrus.Fields, fieldMap FieldMap, reportCaller bool) {
	timeKey := fieldMap.resolve(FieldKeyTime)
	if t, ok := data[timeKey]; ok {
		data["fields."+timeKey] = t
		delete(data, timeKey)
	}

	msgKey := fieldMap.resolve(FieldKeyMsg)
	if m, ok := data[msgKey]; ok {
		data["fields."+msgKey] = m
		delete(data, msgKey)
	}

	levelKey := fieldMap.resolve(FieldKeyLevel)
	if l, ok := data[levelKey]; ok {
		data["fields."+levelKey] = l
		delete(data, levelKey)
	}

	logrusErrKey := fieldMap.resolve(FieldKeyLogrusError)
	if l, ok := data[logrusErrKey]; ok {
		data["fields."+logrusErrKey] = l
		delete(data, logrusErrKey)
	}

	// If reportCaller is not set, 'func' will not conflict.
	if reportCaller {
		funcKey := fieldMap.resolve(FieldKeyFunc)
		if l, ok := data[funcKey]; ok {
			data["fields."+funcKey] = l
		}
		fileKey := fieldMap.resolve(FieldKeyFile)
		if l, ok := data[fileKey]; ok {
			data["fields."+fileKey] = l
		}
	}
}

// Format renders a single log entry
func (f *JSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+4)
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	if f.DataKey != "" {
		newData := make(logrus.Fields, 4)
		newData[f.DataKey] = data
		data = newData
	}

	prefixFieldClashes(data, f.FieldMap, entry.HasCaller())

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}
	//check internal logrus error
	// if entry.err != "" {
	// 	data[f.FieldMap.resolve(FieldKeyLogrusError)] = entry.err
	// }
	if !f.DisableTimestamp {
		data[f.FieldMap.resolve(FieldKeyTime)] = entry.Time.Format(timestampFormat)
	}
	data[f.FieldMap.resolve(FieldKeyLevel)] = entry.Level.String()
	if entry.HasCaller() {
		funcVal := entry.Caller.Function
		fileVal := fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		if f.CallerPrettyfier != nil {
			funcVal, fileVal = f.CallerPrettyfier(entry.Caller)
		}
		if funcVal != "" {
			data[f.FieldMap.resolve(FieldKeyFunc)] = funcVal
		}
		if fileVal != "" {
			data[f.FieldMap.resolve(FieldKeyFile)] = fileVal
		}
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	data[f.FieldMap.resolve(FieldKeyMsg)] = entry.Message
	if !f.DisableMsgJsonOpject {
		var msgObj any
		for _, msg := range []string{entry.Message, "{" + entry.Message + "}", "[" + entry.Message + "]"} {
			if err := json.Unmarshal([]byte(msg), &msgObj); err == nil {
				data[f.FieldMap.resolve(FieldKeyMsg)] = msgObj
				break
			}
		}
	}

	err := ReorderJSONKeys(data, b, f.ReorderArrayKeys, f.DisableHTMLEscape)
	if err != nil {
		return []byte{}, err
	}
	if f.PrettyPrint {
		str, err := ppjson.FormatString(b.String(), "  ")
		if err == nil {
			return []byte(str), nil
		}
	}
	return b.Bytes(), nil
}

func ReorderJSONKeys(data map[string]any, buf io.Writer, keyOrder []string, disableHTMLEscape bool) error {
	if len(keyOrder) == 0 {
		encoder := json.NewEncoder(buf)
		encoder.SetEscapeHTML(!disableHTMLEscape)
		if err := encoder.Encode(data); err != nil {
			return err
		}
		return nil
	}
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	buf.Write([]byte{'\n'})
	buf.Write([]byte{'{'})
	first := true
	processedKeys := make(map[string]bool)

	// Write the keys in the specified order first
	for _, key := range keyOrder {
		if value, exists := data[key]; exists {
			processedKeys[key] = true
			if !first {
				buf.Write([]byte{','})
			}
			first = false

			keyBytes, _ := json.Marshal(key)
			valueBytes, _ := json.Marshal(value)
			buf.Write(keyBytes)
			buf.Write([]byte{':'})
			buf.Write(valueBytes)
		}
	}

	// Collect remaining keys (those in data but not processed yet)
	remainingKeys := make([]string, 0, len(data)-len(processedKeys))
	for key := range data {
		if !processedKeys[key] {
			remainingKeys = append(remainingKeys, key)
		}
	}

	// Sort the remaining keys
	sort.Strings(remainingKeys)

	// Write the sorted remaining keys
	for _, key := range remainingKeys {
		if !first {
			buf.Write([]byte{','})
		}
		first = false

		keyBytes, _ := json.Marshal(key)
		valueBytes, _ := json.Marshal(data[key])
		buf.Write(keyBytes)
		buf.Write([]byte{':'})
		buf.Write(valueBytes)
	}

	buf.Write([]byte{'}'})
	return nil
}
