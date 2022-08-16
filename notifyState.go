package main

import "fmt"

type NotifyMessageState int

const (
	DONE NotifyMessageState = iota
	START
	CMD_RC
	EXIT_RC
	ERROR
	WARN
	EXIT
	SET_LOC
	SET_MEM
	TO_CLIP
	SAVE_EN
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
	return fmt.Sprintf(" error:\"%s\"", nm.err.Error())
}

func (nm *NotifyMessage) getMsg() string {
	if nm.message == "" {
		return ""
	}
	return fmt.Sprintf(" msg:\"%s\"", nm.message)
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
	case CMD_RC:
		return "CMD_RC: "
	case EXIT_RC:
		return "EXIT_RC:"
	case ERROR:
		return "ERROR:  "
	case WARN:
		return "WARN:   "
	case DONE:
		return "DONE:   "
	case START:
		return "START:  "
	case EXIT:
		return "EXIT:   "
	case LOG:
		return "LOG:    "
	case SET_LOC:
		return "SET_LOC:"
	case SET_MEM:
		return "SET_MEM:"
	case TO_CLIP:
		return "TO_CLIP:"
	case SAVE_EN:
		return "SAVE_EN"
	}
	return "??????:"
}

func (nm *NotifyMessage) String() string {
	if nm.action == nil {
		return fmt.Sprintf("Event %s%s%s%s%s", nm.getState(), nm.getMsg(), nm.getNote(), nm.getCode(), nm.getError())
	}
	return fmt.Sprintf("Event %s action:\"%s\"%s%s%s%s", nm.getState(), nm.action.name, nm.getMsg(), nm.getNote(), nm.getCode(), nm.getError())
}
