package main

import "fmt"

type NotifyMessageState int

const (
	DONE NotifyMessageState = iota
	START
	WARN
	ERROR
	EXIT
	LOG
)

type NotifyMessage struct {
	state        NotifyMessageState
	action       *MultipleActionData
	message      string
	notification string
	err          error
	code         int
}

func NewNotifyMessage(state NotifyMessageState, action *MultipleActionData, message, note string, code int, err error) *NotifyMessage {
	return &NotifyMessage{state: state, action: action, message: message, notification: note, code: code, err: err}
}

func (nm *NotifyMessage) getError() string {
	if nm.err == nil {
		return ""
	}
	return fmt.Sprintf(" err:\"%s\"", nm.err.Error())
}

func (nm *NotifyMessage) getNote() string {
	if nm.notification == "" {
		return ""
	}
	return fmt.Sprintf(" note:\"%s\"", nm.notification)
}

func (nm *NotifyMessage) getCode() string {
	if nm.code == 0 {
		return ""
	}
	return fmt.Sprintf(" code:%d", nm.code)
}

func (nm *NotifyMessage) getState() string {
	switch nm.state {
	case WARN:
		return "WARN: "
	case ERROR:
		return "ERROR:"
	case DONE:
		return "DONE: "
	case START:
		return "START:"
	case EXIT:
		return "EXIT: "
	case LOG:
		return "LOG:  "
	}
	return "?????:"
}

func (nm *NotifyMessage) String() string {
	return fmt.Sprintf("Event %s action:\"%s\" msg:\"%s\"%s%s%s", nm.getState(), nm.action.name, nm.message, nm.getNote(), nm.getCode(), nm.getError())
}
