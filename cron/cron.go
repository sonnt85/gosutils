package cron

import (
	"time"
)

type job struct {
	Month, Day, Weekday  int8
	Hour, Minute, Second int8
	Task                 func(time.Time)
}

const ANY = -1 // mod by MDR

var jobs []job

// This function creates a new job that occurs at the given day and the given
// 24hour time. Any of the values may be -1 as an "any" match, so passing in
// a day of -1, the event occurs every day; passing in a second value of -1, the
// event will fire every second that the other parameters match.
func NewCronJob(month, day, weekday, hour, minute, second int8, task func(time.Time)) {
	cj := job{month, day, weekday, hour, minute, second, task}
	jobs = append(jobs, cj)
}

// This creates a job that fires monthly at a given time on a given day.
func NewMonthlyJob(day, hour, minute, second int8, task func(time.Time)) {
	NewCronJob(ANY, day, ANY, hour, minute, second, task)
}

// This creates a job that fires on the given day of the week and time.
func NewWeeklyJob(weekday, hour, minute, second int8, task func(time.Time)) {
	NewCronJob(ANY, ANY, weekday, hour, minute, second, task)
}

// This creates a job that fires daily at a specified time.
func NewDailyJob(hour, minute, second int8, task func(time.Time)) {
	NewCronJob(ANY, ANY, ANY, hour, minute, second, task)
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

func processJobs() {
	for {
		now := time.Now()
		for _, j := range jobs {
			// execute all our cron tasks asynchronously
			if j.Matches(now) {
				go j.Task(now)
			}
		}
		time.Sleep(time.Second)
	}
}

func init() {
	go processJobs()
}
