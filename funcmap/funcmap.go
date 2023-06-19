package funcmap

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	_ "unsafe"

	"github.com/google/uuid"
	"github.com/sonnt85/gosutils/endec"
	"golang.org/x/exp/constraints"
)

var (
	ErrParamsNotAdapted = errors.New("the number of params is not adapted")
)

// constraints.Ordered
type Task[K constraints.Ordered] struct {
	Name   string
	Id     K
	f      reflect.Value
	params []reflect.Value
	// results []reflect.Value
	results []interface{}
	*sync.RWMutex
	*sync.Cond
	done    bool
	ignore  bool
	msg     string
	errCall error
}

func CopyVariable(s interface{}) interface{} {
	a := []interface{}{s}
	b := make([]interface{}, 1)
	copy(b, a)
	return b[0]
}

//go:noinline
func parserParamsFull(f interface{}, params ...interface{}) (paramsValue []reflect.Value, err error) {
	funcValue := reflect.ValueOf(f)

	if funcValue.Kind() != reflect.Func {
		err = errors.New("f is not function.")
		return
	}
	// if (len(params) != funcValue.Type().NumIn()) ||
	if len(params) == funcValue.Type().NumIn()-1 {
		if !funcValue.Type().IsVariadic() {
			err = ErrParamsNotAdapted
		}
	} else if len(params) != funcValue.Type().NumIn() {
		err = ErrParamsNotAdapted
	}
	if err != nil {
		return
	}
	//    value := reflectvar.New(v.(reflect.Type)).Elem().Interface()
	var paraVal reflect.Value
	var needType reflect.Type
	for k := 0; k < funcValue.Type().NumIn(); k++ {
		if k >= len(params) {
			return
		} else if funcValue.Type().IsVariadic() && (k == funcValue.Type().NumIn()-1) && (len(params) == k-1) { //check missing las variadic
			continue
		}
		// param := params[k]

		needType = funcValue.Type().In(k)
		// reflect.Copy(paraVal, reflect.ValueOf(params[k]))
		paraVal = reflect.ValueOf(params[k])

		if paraVal.Type() == needType || needType.Kind() == reflect.Interface {
			paramsValue = append(paramsValue, paraVal)
		} else {
			err = errors.New("Parameter is wrong type. It should be " + needType.String() + " but it's " + paraVal.Type().String())
			return
		}
	}
	// runtime.Gosched()
	// runtime.GC()
	return
}

//go:noinline
func parserParams(f interface{}, params ...interface{}) (paramsValue []reflect.Value, err error) {
	funcValue := reflect.ValueOf(f)

	if funcValue.Kind() != reflect.Func {
		err = errors.New("f is not function")
		return
	}
	// if (len(params) != funcValue.Type().NumIn()) ||
	if len(params) == funcValue.Type().NumIn()-1 {
		if !funcValue.Type().IsVariadic() {
			err = ErrParamsNotAdapted
		}
	} else if len(params) != funcValue.Type().NumIn() {
		err = ErrParamsNotAdapted
	}
	if err != nil {
		return
	}
	//    value := reflectvar.New(v.(reflect.Type)).Elem().Interface()
	var paraVal reflect.Value
	for k := 0; k < funcValue.Type().NumIn(); k++ {
		if k >= len(params) {
			return
		} else if funcValue.Type().IsVariadic() && (k == funcValue.Type().NumIn()-1) && (len(params) == k-1) { //check missing las variadic
			return
		}
		paraVal = reflect.ValueOf(params[k])
		paramsValue = append(paramsValue, paraVal)
	}
	// runtime.Gosched()
	// runtime.GC()
	return
}

func FIDAuto[K constraints.Ordered](fid func() K) func() K {
	if fid == nil {
		fid = func() K { var zero K; return zero }
		var zero K
		var ok bool
		if _, ok = any(zero).(string); ok {
			fid = any(uuid.NewString).(func() K)
		} else if _, ok = any(zero).(uint64); ok {
			fid = any(endec.RandUnt64).(func() K)
		} else if _, ok = any(zero).(uint32); ok {
			fid = any(endec.RandUnt32).(func() K)
		}
	}
	return fid
}

func NewTask[K constraints.Ordered](taskname string, fid func() K, f interface{}, params ...interface{}) (task *Task[K], err error) {
	var paramsValue []reflect.Value
	funcValue := reflect.ValueOf(f)
	paramsValue, err = parserParams(f, params...)
	if err != nil {
		return
	}
	task = &Task[K]{
		Name:    taskname,
		f:       funcValue,
		params:  paramsValue,
		Id:      FIDAuto(fid)(),
		RWMutex: new(sync.RWMutex),
	}
	task.Cond = sync.NewCond(task.RWMutex)
	return task, nil
}

func (t *Task[K]) GetFuncDetail() (f reflect.Value, params []interface{}, results []interface{}, err error) {
	params = make([]interface{}, 0)
	for _, v := range t.params {
		params = append(params, v.Interface())
	}
	return t.f, params, t.results, t.errCall
}

func (t *Task[K]) GCTask() {
	t.params = nil
	t.f = reflect.ValueOf(struct{}{})
}

func (t *Task[K]) Call() (results []interface{}, err error) {
	defer func() {
		t.Lock()
		if r := recover(); r == nil {
			t.results = results
			t.Broadcast()
		} else {
			err = fmt.Errorf("%v", r)
			t.errCall = err
		}
		t.done = true
		t.Unlock()
	}()
	t.RLock()
	params := make([]reflect.Value, len(t.params))
	copy(params, t.params)
	t.RUnlock()
	results = make([]interface{}, 0)
	for _, i := range t.f.Call(params) {
		results = append(results, i.Interface())
	}
	return
}

func (t *Task[K]) ParamsUpdate(params ...interface{}) error {
	paramsValue, err := parserParams(t.f, params...)
	if err != nil {
		return err
	} else {
		t.Lock()
		t.params = paramsValue
		t.Unlock()
		return nil
	}
}

func (t *Task[K]) ResetParasIfFinish() (b bool) {
	if t.IsFinish() {
		t.Lock()
		t.params = nil
		b = true
		t.Unlock()
	}
	return
}

func (t *Task[K]) GetRetValues() (retValue []interface{}, ok bool) {
	t.RLock()
	defer t.RUnlock()
	return t.results, t.done
}

func (t *Task[K]) WaitTaskFinishThenGetValues() (retValue []interface{}) {
	t.Lock()
	for !t.done {
		t.Wait()
	}
	t.Unlock()
	return t.results
}

func (t *Task[K]) IsFinish() (b bool) {
	t.RLock()
	defer t.RUnlock()
	return t.done
}

func (t *Task[K]) IsIgnore() (b bool) {
	t.RLock()
	defer t.RUnlock()
	return t.ignore
}

func (t *Task[K]) SetIgnore(i bool) {
	t.Lock()
	defer t.Unlock()
	t.ignore = i
}

func (t *Task[K]) SetMsg(msg string) {
	t.Lock()
	defer t.Unlock()
	t.msg = msg
}

func (t *Task[K]) GetMsg() string {
	t.RLock()
	defer t.RUnlock()
	return t.msg
}
