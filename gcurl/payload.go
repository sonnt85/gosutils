package gcurl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Payload struct {
	reader        io.Reader
	closer        io.Closer
	contentLength int64
	contentType   string
}

type UploadFile struct {
	Fieldname string
	Filename  string
}

var emptyPayload = new(Payload)

func newPayload(body interface{}) (*Payload, error) {
	if body == nil {
		return emptyPayload, nil
	}

	switch v := body.(type) {
	case *Payload:
		//		fmt.Println("Body type Payload")
		return v, nil
	case Payload:
		return &v, nil
	case string:
		return NewStringPayload(v), nil
	case []byte:
		return NewBytesPayload(v), nil
	case map[string]string:
		return NewFormPayload(v), nil
	case map[string][]string:
		return NewFormPayload(v), nil
	case url.Values:
		return NewFormPayload(v), nil
	}

	// io.reader
	if v, ok := body.(io.Reader); ok {
		return NewReaderPayload(v), nil
	}

	// struct
	t := reflect.TypeOf(body)
	if t.Kind() == reflect.Struct {
		return NewJSONPayload(&body)
	}
	// point to struct
	if t.Kind() == reflect.Ptr && reflect.ValueOf(body).Elem().Kind() == reflect.Struct {
		return NewJSONPayload(body)
	}

	return nil, fmt.Errorf("unsupported payload type: %T", body)
}

func NewStringPayload(body string) *Payload {
	return &Payload{
		reader:        strings.NewReader(body),
		contentLength: int64(len(body)),
	}
}

func NewBytesPayload(body []byte) *Payload {
	return &Payload{
		reader:        bytes.NewReader(body),
		contentLength: int64(len(body)),
	}
}

func NewReaderPayload(reader io.Reader) *Payload {
	return &Payload{
		reader: reader,
	}
}

func NewFilePayload(filename string) (*Payload, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	fstat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return &Payload{
		reader:        f,
		closer:        f,
		contentLength: fstat.Size(),
		contentType:   contentType,
	}, nil
}

func NewJSONPayload(obj interface{}) (*Payload, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return &Payload{
		reader:        bytes.NewReader(body),
		contentLength: int64(len(body)),
		contentType:   "application/json; charset=utf-8",
	}, nil
}

func NewFormPayload(form interface{}) *Payload {
	body := newValues(form).Encode()
	return &Payload{
		reader:        strings.NewReader(body),
		contentLength: int64(len(body)),
		contentType:   "application/x-www-form-urlencoded; charset=utf-8",
	}
}

func NewMultipartPayload(files []UploadFile, form interface{}) (*Payload, error) {
	bodyBuffer := new(bytes.Buffer)
	bodyWriter := multipart.NewWriter(bodyBuffer)
	defer bodyWriter.Close()

	for _, file := range files {
		fileWriter, err := bodyWriter.CreateFormFile(file.Fieldname, file.Filename)
		if err != nil {
			return nil, err
		}

		f, err := os.Open(file.Filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		_, err = io.Copy(fileWriter, f)
		if err != nil {
			return nil, err
		}
	}

	if form != nil {
		for k, vs := range newValues(form) {
			for _, v := range vs {
				bodyWriter.WriteField(k, v)
			}
		}
	}

	return &Payload{
		reader:        bodyBuffer,
		contentLength: int64(bodyBuffer.Len()),
		contentType:   bodyWriter.FormDataContentType(),
	}, nil
}

func newValues(value interface{}) url.Values {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case url.Values:
		return v
	case map[string]string:
		vals := url.Values{}
		for k, v := range v {
			vals.Set(k, v)
		}
		return vals
	case map[string][]string:
		vals := url.Values{}
		for k, vs := range v {
			for _, v := range vs {
				vals.Add(k, v)
			}
		}
		return vals
	}
	return nil
	//	return (fmt.Errorf("unable to convert type %T to url.Values", value))
}
