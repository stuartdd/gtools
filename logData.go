package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type LogData struct {
	logger   *log.Logger
	queue    chan string
	clearLog bool
}

func NewLogData(fileName string, prefix string, clearLog bool) (*LogData, error) {
	lg, err := setup(fileName, prefix, clearLog)
	if err != nil {
		return nil, err
	}

	lg.queue = make(chan string, 20)

	go func(ld *LogData) {
		for l := range lg.queue {
			ld.logger.Println(l)
		}
	}(lg)
	return lg, nil
}

func setup(fileName string, prefix string, clearLog bool) (*LogData, error) {
	if fileName == "" {
		return nil, fmt.Errorf("log file name was not provided")
	}
	flg := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if clearLog {
		flg = os.O_APPEND | os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	}
	file, err := os.OpenFile(fileName, flg, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open file")
	}
	l := log.New(file, prefix, log.Ldate|log.Ltime)
	return &LogData{logger: l, queue: nil}, nil
}

func (lw *LogData) IsLogging() bool {
	return lw.logger != nil && lw.queue != nil
}

func (lw *LogData) Close() {
	if lw.queue == nil {
		lw.logger = nil
		return
	}
	count := 0
	for len(lw.queue) > 0 {
		time.Sleep(500 * time.Millisecond)
		count++
		if count > 20 {
			panic("LogData WitAndClose timed out after 10 seconds!")
		}
	}
	time.Sleep(500 * time.Millisecond)
	close(lw.queue)
	lw.queue = nil
	lw.logger = nil
}

func (lw *LogData) WriteLog(l string) {
	if lw.logger != nil && lw.queue != nil {
		lw.queue <- cleanString(l, 100)
	}
}

func cleanString(s string, max int) string {
	var sb strings.Builder
	count := 0
	for _, r := range s {
		if r < 32 {
			sb.WriteString(fmt.Sprintf("[%d]", r))
		} else {
			sb.WriteRune(r)
		}
		if count >= max {
			break
		}
	}
	return sb.String()
}
