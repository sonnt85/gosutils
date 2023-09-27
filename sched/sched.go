//	 func main() {
//	   jobFunc := func() {
//		     log.Println("Time's up!")
//	   }
//	   sched.Every(5).ESeconds().Run(jobFunc)
//	   sched.Every().DDay().Run(jobFunc)
//	   sched.Every().ESunday().MWDAt("08:30").Run(jobFunc)
//	 }
package sched

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sonnt85/gotimeutils"
)

const (
	EndOfMonth = 0
)

type scheduled interface {
	nextDurationWaitToRun() (time.Duration, error)
	nextRunAt() (time.Time, error)
}

// func (sche scheduled) nextDurationWaitToRun() (time.Duration, error) {

// }

// Job defines a running job and allows to stop a scheduled job or run it.
type Job struct {
	fn        func(fatherJob *Job)
	Quit      chan bool
	skipWait  chan bool
	err       error
	schedule  scheduled
	isRunning bool
	isStart   bool
	sync.RWMutex
}

type recurrent struct {
	units  int
	period time.Duration
	done   bool
}

func (r *recurrent) nextDurationWaitToRun() (time.Duration, error) {
	if r.units == 0 || r.period == 0 {
		return 0, errors.New("cannot set recurrent time with 0")
	}
	if !r.done {
		r.done = true
		return 0, nil
	}
	return time.Duration(r.units) * r.period, nil
}

func (r *recurrent) nextRunAt() (t time.Time, err error) {
	var d time.Duration
	d, err = r.nextDurationWaitToRun()
	if err != nil {
		return
	}
	return time.Now().Add(d), nil
}

type daily struct {
	hour int
	min  int
	sec  int
}

func (d *daily) setTime(h, m, s int) {
	d.hour = h
	d.min = m
	d.sec = s
}

func (d *daily) nextDurationWaitToRun() (du time.Duration, err error) {
	var date time.Time
	date, err = d.nextRunAt()
	if err != nil {
		return
	}
	return time.Until(date), nil
}

func (d *daily) nextRunAt() (time.Time, error) {
	now := time.Now()
	year, month, day := now.Date()
	date := time.Date(year, month, day, d.hour, d.min, d.sec, 0, time.Local)
	if now.Before(date) {
		return date, nil
	}
	date = time.Date(year, month, day+1, d.hour, d.min, d.sec, 0, time.Local)
	return date, nil
}

type weekly struct {
	day time.Weekday
	d   daily
}

func (w *weekly) nextDurationWaitToRun() (du time.Duration, err error) {
	var date time.Time
	date, err = w.nextRunAt()
	if err != nil {
		return
	}
	return time.Until(date), nil
}

func (w *weekly) nextRunAt() (time.Time, error) {
	now := time.Now()
	year, month, day := now.Date()
	numDays := w.day - now.Weekday()
	if numDays == 0 {
		date := time.Date(year, month, day, w.d.hour, w.d.min, w.d.sec, 0, time.Local)
		if now.After(date) {
			numDays = 7
		}
	} else if numDays < 0 {
		numDays += 7
	}
	date := time.Date(year, month, day+int(numDays), w.d.hour, w.d.min, w.d.sec, 0, time.Local)
	return date, nil
}

type monthly struct {
	day int
	d   daily
	// nextTimeRun time.Time
}

func (m *monthly) nextRunAt() (time.Time, error) {
	now := time.Now()
	year, month, day := now.Date()
	numDays := 0
	numDayOfThisMonth := gotimeutils.EndOfMonth().Day()
	if m.day == EndOfMonth {
		if day == numDayOfThisMonth {
			date := time.Date(year, month, day, m.d.hour, m.d.min, m.d.sec, 0, time.Local)
			if now.After(date) {
				numDays = gotimeutils.EndOfNextMonth().Day()
			}
		} else {
			numDays = numDayOfThisMonth - day
		}
	} else {
		deltalDays := m.day - day
		if deltalDays == 0 {
			date := time.Date(year, month, day, m.d.hour, m.d.min, m.d.sec, 0, time.Local)
			if now.After(date) {
				numDays = numDayOfThisMonth - day + m.day
			}
		} else if deltalDays < 0 {
			numDays = numDayOfThisMonth - day + m.day
		}
	}

	date := time.Date(year, month, day+int(numDays), m.d.hour, m.d.min, m.d.sec, 0, time.Local)
	return date, nil
}

func (m *monthly) nextDurationWaitToRun() (du time.Duration, err error) {
	var date time.Time
	date, err = m.nextRunAt()
	if err != nil {
		return
	}
	return time.Until(date), nil
}

// config day of monthly. can run mutiple time
func (j *Job) MDay(d int) *Job {
	if j.schedule != nil {
		if m, ok := j.schedule.(*monthly); ok {
			m.day = d
		} else {
			j.err = errors.New("bad function chaining")
		}
	} else {
		j.schedule = &monthly{day: d}
	}
	return j
}

// Every defines when to run a job. For a recurrent jobs (times seconds/minutes/hours) you
// should specify the unit and then call to the correspondent period method.
func Every(times ...int) *Job {
	switch len(times) {
	case 0:
		return &Job{}
	default:
		r := new(recurrent)
		r.units = times[0]
		return &Job{schedule: r}
	}
}

// RNotImmediately allows recurrent jobs not to be executed immediatelly after
// definition. If a job is declared hourly won't start executing until the first hour
// passed.
func (j *Job) RNotImmediately() *Job {
	j.Lock() //lock for wrire rj.done
	defer j.Unlock()
	rj, ok := j.schedule.(*recurrent)

	if !ok {
		j.err = errors.New("bad function chaining")
		return j
	}
	rj.done = true
	return j
}

// At lets you define a specific time when the job would be run. Does not work with
// recurrent jobs, work with daily or weekly
// Time should be defined as a string separated by a colon. Could be used as "08:35:30",
// "08:35" or "8" for only the hours.
func (j *Job) At(hourTime string) *Job {
	if j.err != nil {
		return j
	}
	hour, min, sec, err := parseTime(hourTime)
	if err != nil {
		j.err = err
		return j
	}
	j.Lock()
	defer j.Unlock()
	if d, ok := j.schedule.(*daily); ok {
		d.setTime(hour, min, sec)
		j.schedule = d
	} else {
		if w, ok := j.schedule.(*weekly); ok {
			w.d.setTime(hour, min, sec)
			j.schedule = w
		} else {
			if m, ok := j.schedule.(*monthly); ok {
				m.d.setTime(hour, min, sec)
			} else {
				j.err = errors.New("bad function chaining")
				return j
			}
		}
	}
	return j
}

// update every times daily
func (j *Job) Every(times ...int) (units int, job *Job, err error) {
	job = j
	j.Lock()
	defer j.Unlock()
	switch len(times) {
	case 0:
		r, ok := j.schedule.(*recurrent)
		if ok {
			units = r.units
		} else {
			err = fmt.Errorf("job is not recurrent")
		}
		return
	case 1:
		r, ok := j.schedule.(*recurrent)
		if ok {
			units = times[0]
			r.units = times[0]
		} else {
			err = fmt.Errorf("job is not recurrent")
		}
		return
	default:
		err = errors.New("too many arguments in Every")
		return
	}
}

func (j *Job) SkipIfWait() {
	if !j.IsRunning() {
		j.skipWait <- true
	}
}

// Run sets the job to the schedule and returns the pointer to the job so it may be
// stopped or executed without waiting or an error.
func (j *Job) Run(fs ...func(*Job)) (*Job, error) {
	if j.err != nil {
		return nil, j.err
	}
	if len(fs) == 0 {
		if j.fn == nil {
			return j, fmt.Errorf("missing function")
		}
	} else {
		j.fn = fs[0]
	}
	if j.isStart {
		return j, fmt.Errorf("already start")
	}
	j.isStart = true
	var next time.Duration
	var err error
	j.Quit = make(chan bool, 1)
	j.skipWait = make(chan bool, 1)
	// Check for possible errors in scheduling
	// next, err = j.schedule.nextDurationWaitToRun()
	if err != nil {
		return nil, err
	}

	go func(j *Job) {
		for {
			select {
			case <-j.Quit:
				return
			case <-j.skipWait:
				go runJob(j)
			case <-time.After(next):
				go runJob(j)
			}
			j.RLock()
			next, _ = j.schedule.nextDurationWaitToRun()
			j.RUnlock()
		}
	}(j)
	return j, nil
}

func (j *Job) SetFunc(f func(*Job)) *Job {
	if j.err != nil {
		return j
	}
	j.Lock()
	if j.Quit == nil {
		j.Quit = make(chan bool, 1)
		j.skipWait = make(chan bool, 1)
	}
	j.fn = f
	j.Unlock()
	return j
}

// need call SetFunc before call this function
func (j *Job) runCheck() error {
	if j.fn == nil {
		return fmt.Errorf("Need call SetFunc before call this function")
	}
	if j.err != nil {
		return j.err
	}
	if j.isStart {
		return fmt.Errorf("Already start")
	}
	j.isStart = true
	var next time.Duration
	var err error
	// Check for possible errors in scheduling
	next, err = j.schedule.nextDurationWaitToRun()
	if err != nil {
		return err
	}
	select {
	case <-time.After(next):
		go runJob(j)
	}
	return nil
}

func (j *Job) setRunning(running bool) {
	j.Lock()
	defer j.Unlock()

	j.isRunning = running
}

func runJob(job *Job) {
	if job.IsRunning() {
		return
	}
	job.setRunning(true)
	job.fn(job)
	job.setRunning(false)
}

func parseTime(str string) (hour, min, sec int, err error) {
	chunks := strings.Split(str, ":")
	var hourStr, minStr, secStr string
	switch len(chunks) {
	case 1:
		hourStr = chunks[0]
		minStr = "0"
		secStr = "0"
	case 2:
		hourStr = chunks[0]
		minStr = chunks[1]
		secStr = "0"
	case 3:
		hourStr = chunks[0]
		minStr = chunks[1]
		secStr = chunks[2]
	}
	hour, err = strconv.Atoi(hourStr)
	if err != nil {
		return 0, 0, 0, errors.New("bad time")
	}
	min, err = strconv.Atoi(minStr)
	if err != nil {
		return 0, 0, 0, errors.New("bad time")
	}
	sec, err = strconv.Atoi(secStr)
	if err != nil {
		return 0, 0, 0, errors.New("bad time")
	}

	if hour > 23 || min > 59 || sec > 59 {
		return 0, 0, 0, errors.New("bad time")
	}

	return
}

// Weekly schedule  (Sunday = 0, ...).
func (j *Job) dayOfWeek(d time.Weekday) *Job {
	if j.schedule != nil {
		j.err = errors.New("bad function chaining")
	}
	j.schedule = &weekly{day: d}
	return j
}

// WMonday sets the job to run every WMonday.
func (j *Job) WMonday() *Job {
	return j.dayOfWeek(time.Monday)
}

// WTuesday sets the job to run every WTuesday.
func (j *Job) WTuesday() *Job {
	return j.dayOfWeek(time.Tuesday)
}

// WWednesday sets the job to run every WWednesday.
func (j *Job) WWednesday() *Job {
	return j.dayOfWeek(time.Wednesday)
}

// WThursday sets the job to run every WThursday.
func (j *Job) WThursday() *Job {
	return j.dayOfWeek(time.Thursday)
}

// WFriday sets the job to run every WFriday.
func (j *Job) WFriday() *Job {
	return j.dayOfWeek(time.Friday)
}

// WSaturday sets the job to run every WSaturday.
func (j *Job) WSaturday() *Job {
	return j.dayOfWeek(time.Saturday)
}

// WSunday sets the job to run every WSunday.
func (j *Job) WSunday() *Job {
	return j.dayOfWeek(time.Sunday)
}

// DDay sets the job  is daily [run every day (daily - h:m:s).]
func (j *Job) DDay() *Job {
	if j.schedule != nil {
		if _, ok := j.schedule.(*daily); !ok {
			j.err = errors.New("bad function chaining")
		}
	}
	j.schedule = &daily{}
	return j
}

func (j *Job) timeOfDay(d time.Duration) *Job {
	if j.err != nil {
		return j
	}
	r := j.schedule.(*recurrent)
	r.period = d
	j.schedule = r
	return j
}

// Seconds sets the job to run every n Seconds where n was defined in the Every
// function.
func (j *Job) ESeconds() *Job {
	return j.timeOfDay(time.Second)
}

// EMinutes sets the job to run every n EMinutes where n was defined in the Every
// function.
func (j *Job) EMinutes() *Job {
	return j.timeOfDay(time.Minute)
}

// EHours sets the job to run every n EHours where n was defined in the Every function.
func (j *Job) EHours() *Job {
	return j.timeOfDay(time.Hour)
}

// IsRunning returns if the job is currently running
func (j *Job) IsRunning() bool {
	j.RLock()
	defer j.RUnlock()
	return j.isRunning
}
