package logs

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

/**
* This code is basically a copy of gorm.logger (that struct is private)
* Difference is that we don't print info messages or check for loglevel
* And we don't print record not found errors because these are intended most of the time
 */

type databaseLogger struct {
	logger.Writer
	SlowThreshold                                        time.Duration
	warnStr, errStr, traceStr, traceErrStr, traceWarnStr string
}

// LogMode does nothing
func (l *databaseLogger) LogMode(level logger.LogLevel) logger.Interface {
	newlogger := *l
	return &newlogger
}

// Info does nothing
func (l databaseLogger) Info(ctx context.Context, msg string, data ...interface{}) {}

// Warn print warn messages
func (l databaseLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.Printf(l.warnStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
}

// Error print error messages
func (l databaseLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.Printf(l.errStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
}

// Trace print sql message
func (l databaseLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil:
		// Don't print record not found errors
		if strings.Contains(err.Error(), "record not found") {
			return
		}

		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.Printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

// NewDatabaseLogger returns a logger based on the environment
func NewDatabaseLogger(logFile *os.File) logger.Interface {
	// Set up logging so it writes to stdout and to a file
	wrt := io.MultiWriter(os.Stdout, logFile)
	writer := log.New(wrt, "\r\n", log.LstdFlags)
	slow := time.Duration(time.Millisecond * 50)

	// If production we log all messages excluding info, and not found errors
	if os.Getenv("ENV") == "production" {
		warnStr := "%s\n[warn] "
		errStr := "%s\n[error] "
		traceStr := "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr := "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr := "%s %s\n[%.3fms] [rows:%v] %s"

		return &databaseLogger{
			Writer:        writer,
			SlowThreshold: slow,
			warnStr:       warnStr,
			errStr:        errStr,
			traceStr:      traceStr,
			traceWarnStr:  traceWarnStr,
			traceErrStr:   traceErrStr,
		}
	}

	// Log basically everything
	return logger.New(writer, logger.Config{
		SlowThreshold: slow,
		LogLevel:      logger.Info,
	})
}
