package slogrus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	filepath "github.com/sonnt85/gofilepath"

	"github.com/sonnt85/gosutils/endec"
)

// RotateLogsCheck performs checks and management of log files for an application.
// This function verifies the maximum log file size (maxFileSize) and the maximum number of backups (maxBackups).
//
// The maxAge parameter has the following significance:
// - If maxAge is an integer (int), it represents the maximum number of days a log file can exist before being deleted.
// - If maxAge is a time duration value (time.Duration), it represents the maximum time duration a log file can exist before being deleted.
//
// The logFiles parameter is a list of log files to be checked and managed.
//
// This function returns an error (err) if any errors occur during the process of checking and managing log files.
func RotateLogsCheck(maxFileSize, maxBackups int, maxAge interface{}, logFiles ...string) (err error) {
	for _, logFile := range logFiles {
		err = errors.Join(err, RotateLogCheck(logFile, maxFileSize, maxBackups, maxAge))
	}
	return
}

// RotateLogsMonitor continuously monitors and manages log files for an application at regular intervals.
// It accepts a context (ctx) to control the monitoring lifecycle, a monitoring period (period), maximum log file size (maxFileSize),
// maximum number of backups (maxBackups), and a maxAge parameter.
//
// The maxAge parameter has the following significance:
// - If maxAge is an integer (int), it represents the maximum number of days a log file can exist before being deleted.
// - If maxAge is a time duration value (time.Duration), it represents the maximum time duration a log file can exist before being deleted.
//
// The logFiles parameter is a list of log files to be monitored and managed.
//
// This function operates in a loop, periodically checking log files and performing necessary log rotation actions based on the specified parameters.
// It can be controlled and terminated using the provided context (ctx).
func RotateLogsMonitor(ctx context.Context, period time.Duration, maxFileSize, maxBackups int, maxAge interface{}, logFiles ...string) {
	if len(logFiles) == 0 {
		return
	}
	ticker := time.NewTicker(period)
	if ctx == nil {
		ctx, _ = context.WithCancel(context.Background())
	}
	go func() {
		for {
			select {
			case <-ticker.C:
				RotateLogsCheck(maxFileSize, maxBackups, maxAge, logFiles...)

			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// RotateLogCheck checks and manages a single log file for an application.
// It verifies the log file's size against the maximum allowed size (maxFileSize),
// the number of backups against the maximum allowed backups (maxBackups), and
// manages the log file's age based on the provided maxAge parameter.
//
// The maxAge parameter has the following significance:
// - If maxAge is an integer (int), it represents the maximum number of days the log file can exist before being deleted.
// - If maxAge is a time duration value (time.Duration), it represents the maximum time duration the log file can exist before being deleted.
//
// The logFile parameter specifies the path to the log file that needs to be checked and managed.
//
// This function returns an error (err) if any errors occur during the process of checking and managing the log file.
func RotateLogCheck(logFile string, maxFileSize, maxBackups int, maxAge interface{}) (err error) {
	// Check the size of the log file
	var fileInfo fs.FileInfo
	fileInfo, err = os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			// The log file does not exist, does not need to turn around
			return nil
		}
		return fmt.Errorf("failed to get file info: %v", err)
	}
	fileSize := int(fileInfo.Size()) /// 1024 // Size conversion to KB
	if fileSize <= maxFileSize {
		// Log file size has not exceeded the limit, no need to rotate
		return nil
	}
	// Create a new backup file name
	backupFile := backupName(logFile, true)

	err = copyFile(logFile, backupFile)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %v", err)
	}

	// Truncate origin file
	err = os.Truncate(logFile, 0)
	if err != nil {
		return fmt.Errorf("failed to truncate log file: %v", err)
	}

	// delete old backup files if exceeding the maximum number of backups
	logFileDir := filepath.Dir(logFile)
	var compress, remove []logInfo

	// Filter and delete old backup files if it exceeds the maximum number of backups
	files, err := getOldLogFiles(logFile)

	if err != nil || len(files) == 0 {
		return err
	}

	if maxBackups > 0 && maxBackups < len(files) {
		preserved := make(map[string]bool)
		var remaining []logInfo
		for _, f := range files {
			// Only count the uncompressed log file or the
			// compressed log file, not both.
			fn := f.Name()
			if strings.HasSuffix(fn, compressSuffix) {
				fn = fn[:len(fn)-len(compressSuffix)]
			}
			preserved[fn] = true

			if len(preserved) > maxBackups {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}
	var maxAgeDuration time.Duration
	switch maxAge.(type) {
	case int:
		days := maxAge.(int)
		maxAgeDuration = time.Duration(days) * 24 * time.Hour
	case time.Duration:
		maxAgeDuration = maxAge.(time.Duration)
	}

	if maxAgeDuration > 0 {
		// diff := time.Duration(int64(24*time.Hour) * int64(maxAge))
		cutoff := currentTime().Add(-1 * maxAgeDuration)

		var remaining []logInfo
		for _, f := range files {
			if f.timestamp.Before(cutoff) {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}

	if true {
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), compressSuffix) {
				compress = append(compress, f)
			}
		}
	}

	for _, f := range remove {
		errRemove := os.Remove(filepath.Join(logFileDir, f.Name()))
		if err == nil && errRemove != nil {
			err = errRemove
		}
	}

	for _, f := range compress {
		fn := filepath.Join(logFileDir, f.Name())
		zipname := fn + compressSuffix
		info, errinfo := os_Stat(fn)
		errCompress := endec.GzipFile(zipname, fn, true, -1)
		if errCompress == nil && zipPostHook != nil {
			zipPostHook(zipname)
		}
		if err == nil && errCompress != nil {
			err = errCompress
		}
		if err == nil && errinfo == nil {
			if errinfo = chown(zipname, info); errinfo != nil {
				err = errinfo
			}
		}
	}
	return err
}

func copyFile(srcFile, destFile string) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer dest.Close()

	if _, err = io.Copy(dest, src); err == nil {
		if info, errinfo := os_Stat(srcFile); errinfo == nil {
			if err = chown(destFile, info); err == nil {
				err = os.Chmod(destFile, info.Mode())
			}
		}
	}
	return err
}

func prefixAndExt(filename string) (prefix, ext string) {
	filename = filepath.Base(filename)
	ext = filepath.Ext(filename)
	prefix = filename[:len(filename)-len(ext)] + "-"
	return prefix, ext
}

func timeFromName(filename, prefix, ext string) (time.Time, error) {
	if !strings.HasPrefix(filename, prefix) {
		return time.Time{}, errors.New("mismatched prefix")
	}
	if !strings.HasSuffix(filename, ext) {
		return time.Time{}, errors.New("mismatched extension")
	}
	ts := filename[len(prefix) : len(filename)-len(ext)]
	return time.Parse(backupTimeFormat, ts)
}

func getOldLogFiles(logFile string) ([]logInfo, error) {
	files, err := os.ReadDir(filepath.Dir(logFile))
	if err != nil {
		return nil, fmt.Errorf("can't read log file directory: %s", err)
	}
	logFiles := []logInfo{}

	prefix, ext := prefixAndExt(logFile)
	var t time.Time

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if t, err = timeFromName(f.Name(), prefix, ext); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
			continue
		}
		if t, err = timeFromName(f.Name(), prefix, ext+compressSuffix); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
			continue
		}
		// error parsing means that the suffix at the end was not generated
		// by lumberjack, and therefore it's not a backup file.
	}

	sort.Sort(byFormatTime(logFiles))

	return logFiles, nil
}

func getSubstring(str1, str2 string) string {
	dotCount := strings.Count(str1, ".")
	str2Parts := strings.Split(str2, ".")
	if len(str2Parts) == dotCount+2 || len(str2Parts) == dotCount+3 {
		return strings.Join(str2Parts[:dotCount+1], ".")
	}
	return str2
}

func notMatchLogBackup(filename string) bool {
	if lastIndex := strings.LastIndex(filename, "-"); lastIndex == -1 {
		return true
	} else {
		ts := filename[lastIndex+1:]
		// ts = strings.TrimSuffix(ts, filepath.Ext(ts))
		ts = getSubstring(backupTimeFormat, ts)
		_, err := time.Parse(backupTimeFormat, ts)
		return err != nil
	}
}

// maxdeep from 0
// patternRegex is regexp
func GetOriginLogFiles(dir, patternRegex string, maxdeep int) (logFiles []string) {
	files := filepath.FindFilesMatchRegexpName(dir, patternRegex, maxdeep, true, false)
	for _, f := range files {
		if notMatchLogBackup(f) {
			logFiles = append(logFiles, f)
			continue
		}
	}
	sort.Strings(logFiles)
	return logFiles
}

// RotateDirsCheck checks and manages log directories for an application.
// It verifies the maximum log file size (maxFileSize), the maximum number of backups (maxBackups),
// and manages log files within the specified directories based on the provided maxAge parameter.
//
// The maxAge parameter has the following significance:
// - If maxAge is an integer (int), it represents the maximum number of days log files within the monitored directories can exist before being deleted.
// - If maxAge is a time duration value (time.Duration), it represents the maximum time duration log files within the monitored directories can exist before being deleted.
//
// The patternRegex parameter specifies a regular expression pattern used to match log files within the monitored directories.
// The maxdeep parameter defines the maximum directory depth to search for log files.
//
// The logDirs parameter is a list of directory paths to be checked and managed.
//
// This function returns an error (err) if any errors occur during the process of checking and managing log directories.
func RotateDirsCheck(maxFileSize, maxBackups int, maxAge interface{}, patternRegex string, maxdeep int, logDirs ...string) (err error) {
	for _, dir := range logDirs {
		for _, logFile := range GetOriginLogFiles(dir, patternRegex, maxdeep) {
			// fmt.Printf("checking file: %s\n", logFile)
			err = errors.Join(err, RotateLogCheck(logFile, maxFileSize, maxBackups, maxAge))
		}
	}
	return
}

// RotateDirsMonitor continuously monitors and manages log directories for an application at regular intervals.
// It accepts a context (ctx) to control the monitoring lifecycle, a monitoring period (period), maximum log file size (maxFileSize),
// maximum number of backups (maxBackups), and a maxAge parameter.
//
// The maxAge parameter has the following significance:
// - If maxAge is an integer (int), it represents the maximum number of days log files within the monitored directories can exist before being deleted.
// - If maxAge is a time duration value (time.Duration), it represents the maximum time duration log files within the monitored directories can exist before being deleted.
//
// The patternRegex parameter specifies a regular expression pattern used to match log files within the monitored directories.
// The maxdeep parameter defines the maximum directory depth to search for log files.
//
// The logDirs parameter is a list of directory paths to be monitored and managed.
//
// This function operates in a loop, periodically checking log directories and performing necessary log rotation actions based on the specified parameters.
// It can be controlled and terminated using the provided context (ctx).
func RotateDirsMonitor(ctx context.Context, period time.Duration, maxFileSize, maxBackups int, maxAge interface{}, patternRegex string, maxdeep int, logDirs ...string) {
	if len(logDirs) == 0 {
		return
	}
	ticker := time.NewTicker(period)
	if ctx == nil {
		ctx, _ = context.WithCancel(context.Background())
	}
	go func() {
		for {
			select {
			case <-ticker.C:
				RotateDirsCheck(maxFileSize, maxBackups, maxAge, patternRegex, maxdeep, logDirs...)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
