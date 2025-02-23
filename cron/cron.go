package cron

import (
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/sonnt85/gosutils/endec"
)

const ANY = -1 // mod by MDR

type job struct {
	Month, Day, Weekday  int8
	Hour, Minute, Second int8
	Task                 func(time.Time)
	Name                 string
	IsRunning            bool
}

func (cj job) Matches(t time.Time) (ok bool) {
	ok = (cj.Month == ANY || cj.Month == int8(t.Month())) &&
		(cj.Day == ANY || cj.Day == int8(t.Day())) &&
		(cj.Weekday == ANY || cj.Weekday == int8(t.Weekday())) &&
		(cj.Hour == ANY || cj.Hour == int8(t.Hour())) &&
		(cj.Minute == ANY || cj.Minute == int8(t.Minute())) &&
		(cj.Second == ANY || cj.Second == int8(t.Second()))
	return ok
}

func (cj job) nextMatches() (t time.Time) {
	// var month, day, weekday int8
	// var hour, minute, second int8
	// if cj.Month != ANY {
	// 	month = cj.Month
	// }
	// if cj.Day != ANY {
	// 	day = j.Day
	// }
	// if cj.Weekday != ANY {
	// 	weekday = cj.Weekday
	// }
	// if cj.Hour != ANY {
	// 	hour = cj.Hour
	// }
	// if cj.Minute != ANY {
	// 	minute = cj.Minute
	// }
	// if cj.Second != ANY {
	// 	second = cj.Second
	// }
	return time.Now()
}

type Jobs struct {
	// tiker time.Ticker
	sync.RWMutex
	J *[](job)
}

var jobs *Jobs //global stored jobs

func (js *Jobs) Len() int      { return len(*js.J) }
func (js *Jobs) Swap(i, j int) { (*js.J)[i], (*js.J)[j] = (*js.J)[j], (*js.J)[i] }
func (js *Jobs) Less(i, j int) bool {
	ji := (*js.J)[i]
	jj := (*js.J)[j]
	// return ji.Month < jj.Month || ji.Day < jj.Day || ji.Hour < jj.Hour || ji.Minute < jj.Minute || ji.Second < jj.Second
	return ji.Month < jj.Month || ji.Day < jj.Day || ji.Weekday < jj.Weekday || ji.Hour < jj.Hour || ji.Minute < jj.Minute || ji.Second < jj.Second
}

func (js *Jobs) Truncate(n int) { *js.J = (*js.J)[:n] }

// This function creates a new job that occurs at the given day and the given
// 24hour time. Any of the values may be -1 as an "any" match, so passing in

// a day of -1, the event occurs every day; passing in a second value of -1, the
// event will fire every second that the other parameters match.
func NewCronJob(month, day, weekday, hour, minute, second int8, task func(time.Time), tasknames ...string) (taskid string) {
	if jobs == nil {
		return
	}
	taskname := endec.GenerateRandomAssci(16)
	if len(tasknames) != 0 {
		taskname = tasknames[0] + "-" + taskname
	}
	cj := job{month, day, weekday, hour, minute, second, task, taskname, false}
	jobs.Lock()
	*jobs.J = append(*jobs.J, cj)
	sort.Sort(jobs)
	jobs.Unlock()
	return taskname
}

// This creates a job that fires monthly at a given time on a given day.
func NewMonthlyJob(day, hour, minute, second int8, task func(time.Time), taskname ...string) (taskid string) {
	return NewCronJob(ANY, day, ANY, hour, minute, second, task, taskname...)
}

// This creates a job that fires on the given day of the week and time.
func NewWeeklyJob(weekday, hour, minute, second int8, task func(time.Time), taskname ...string) (taskid string) {
	return NewCronJob(ANY, ANY, weekday, hour, minute, second, task, taskname...)
}

// This creates a job that fires daily at a specified time.
func NewDailyJob(hour, minute, second int8, task func(time.Time), taskname ...string) (taskid string) {
	return NewCronJob(ANY, ANY, ANY, hour, minute, second, task, taskname...)
}

// DeleteTask deletes a task with the specified task ID from the jobs slice.
// It returns true if the task is successfully deleted, false otherwise.
func DeleteTask(taskid string) bool {
	jobs.Lock()
	defer jobs.Unlock()

	for index, jTemp := range *jobs.J {
		if jTemp.Name == taskid {
			*jobs.J = slices.Delete(*jobs.J, index, index)
			// if index == 0 {
			// 	*jobs.J = (*jobs.J)[1:]
			// }
			// if index == len(*jobs.J)-1 {
			// 	*jobs.J = (*jobs.J)[:len(*jobs.J)-1]
			// 	return true
			// }
			// *jobs.J = append((*jobs.J)[:index], (*jobs.J)[index+1:]...)
			return true
		}
	}
	return false
}

// UpdateTask updates the attributes of a task with the specified task ID in the jobs slice.
// It returns true if the task is successfully updated, false otherwise.
// Update the task's attributes if the provided value is not -2.
func UpdateTask(taskid string, month, day, weekday, hour, minute, second int8) bool {
	jobs.Lock()
	defer jobs.Unlock()
	for i, jTemp := range *jobs.J {
		if jTemp.Name == taskid {
			if month != -2 {
				jTemp.Month = month
			}
			if month != -2 {
				jTemp.Day = day
			}
			if weekday != -2 {
				jTemp.Weekday = weekday
			}
			if hour != -2 {
				jTemp.Hour = hour
			}
			if minute != -2 {
				jTemp.Minute = minute
			}
			if second != -2 {
				jTemp.Second = second
			}
			(*jobs.J)[i] = jTemp
			return true
		}
	}
	return false
}

// NewCronJob(month, day, weekday, hour, minute, second

func processJobs() {
	for {
		now := time.Now()
		nextSecond := now.Truncate(time.Second).Add(time.Second)
		jobs.RLock()
		for _, jTemp := range *jobs.J {
			// execute all our cron tasks asynchronously
			j := jTemp
			if j.Matches(now) && !j.IsRunning {
				j.IsRunning = true
				go func() {
					j.Task(now)
					j.IsRunning = false
				}()
			}
		}
		jobs.RUnlock()
		time.Sleep(time.Until(nextSecond))
	}
}

func InitCron() {
	jobs = new(Jobs)
	jbs := make([]job, 0)
	jobs.J = &jbs
	go processJobs()
}
