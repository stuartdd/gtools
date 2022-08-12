package main

import (
	"fmt"
	"io"
	"os/exec"
	"time"
)

type ActionState int

const (
	RC_SETUP = -1
	RC_CLEAN = 0
	RC_ERROR = 1

	DONE ActionState = iota
	START
	WARN
	ERROR
	EXIT
)

func execDelayedAction(action *MultipleActionData, delay int, notifyActionState func(ActionState, string, string, error) int, dataCache *DataCache) {
	if debugLogMain.IsLogging() {
		debugLogMain.WriteLog(fmt.Sprintf("Run action \"%s\" delayed %d ms", action, delay))
	}
	go func() {
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		} else {
			time.Sleep(time.Duration(100 * time.Millisecond))
		}
		execMultipleAction(action, notifyActionState, dataCache)
	}()
}

func execMultipleAction(data *MultipleActionData, notifyActionState func(ActionState, string, string, error) int, dataCache *DataCache) {
	if notifyActionState != nil {
		notifyActionState(START, data.name, "", nil)
	}
	if debugLogMain.IsLogging() {
		debugLogMain.WriteLog("  Started " + data.String())
	}
	defer func() {
		if notifyActionState != nil {
			notifyActionState(DONE, data.name, "", nil)
		}
		if debugLogMain.IsLogging() {
			debugLogMain.WriteLog("  Ended " + data.String())
		}
	}()

	stdOut := NewBaseWriter("", stdColourPrefix[STD_OUT])
	stdErr := NewBaseWriter("", stdColourPrefix[STD_ERR])
	for i, act := range data.commands {
		locationMsg := fmt.Sprintf("Action '%s' step '%d' path '%s'", data.desc, i, act.Dir())
		rc, err := execSingleAction(act, stdOut, stdErr, data.desc, dataCache)
		if err != nil {
			if rc == RC_SETUP {
				if debugLogMain.IsLogging() {
					debugLogMain.WriteLog(fmt.Sprintf("    Error Setup: %s. %s ", err.Error(), act.String()))
				}
				if notifyActionState != nil {
					notifyActionState(WARN, locationMsg, "", err)
				}
				return
			}
			if act.ignoreError {
				if debugLogMain.IsLogging() {
					debugLogMain.WriteLog(fmt.Sprintf("    Error Ignored: %s. %s ", err.Error(), act.String()))
				}
			} else {
				if debugLogMain.IsLogging() {
					debugLogMain.WriteLog(fmt.Sprintf("    Error: %s. %s ", err.Error(), act.String()))
				}
				exitOsMsg := fmt.Sprintf("Exit to OS with RC=%d", rc)
				if notifyActionState != nil {
					resp := notifyActionState(WARN, locationMsg, exitOsMsg, err)
					if resp == 1 {
						exitApp(fmt.Sprintf("%s. RC[%d] Error:%s", locationMsg, rc, err.Error()), rc)
					}
				}
				return
			}
		}
		if debugLogMain.IsLogging() {
			debugLogMain.WriteLog(fmt.Sprintf("    Command: path:\"%s\" rc:%d cmd:\"%s %s\"", act.Dir(), rc, act.command, act.args))
		}
	}
	if data.rc >= 0 {
		exitApp("", data.rc)
	}
}

func execSingleAction(sa *SingleAction, stdOut, stdErr *BaseWriter, actionDesc string, dataCache *DataCache) (int, error) {
	outEncKey, err := derivePasswordFromName(sa.outPwName, sa, dataCache)
	if err != nil {
		return RC_SETUP, err
	}
	inEncKey, err := derivePasswordFromName(sa.inPwName, sa, dataCache)
	if err != nil {
		return RC_SETUP, err
	}
	args, err := substituteValuesIntoArgs(sa.args, validatedEntryDialog, dataCache)
	if err != nil {
		return RC_SETUP, err
	}
	cmd := exec.Command(sa.command, args...)
	if sa.directory != "" {
		cmd.Dir = sa.directory
	}
	if sa.sysinDef != "" {
		tmp, err := substituteValuesIntoString(sa.sysinDef, sysInDialog, dataCache)
		if err != nil {
			return RC_SETUP, err
		}
		si, err := NewStringReader(tmp, cmd.Stdin, dataCache)
		if err != nil {
			return RC_SETUP, err
		}
		siCloser, ok := si.(io.ReadCloser)
		if ok {
			defer siCloser.Close()
		}
		encR, ok := si.(EncReader)
		if ok {
			encR.SetKey(inEncKey)
		}
		cmd.Stdin = si
	}
	sysoutDef, err := substituteValuesIntoString(sa.sysoutDef, sysOutDialog, dataCache)
	if err != nil {
		return RC_SETUP, err
	}
	so := NewWriter(sysoutDef, outEncKey, stdOut, stdErr, dataCache)
	soReset, reSoOk := so.(Reset)
	if reSoOk {
		soReset.Reset()
	}
	soCloser, soOk := so.(io.Closer)
	if soOk {
		defer soCloser.Close()
	}
	cmd.Stdout = so

	syserrDef, err := substituteValuesIntoString(sa.syserrDef, sysOutDialog, dataCache)
	if err != nil {
		return RC_SETUP, err
	}
	se := NewWriter(syserrDef, outEncKey, stdErr, stdErr, dataCache)
	seReset, reSeOk := se.(Reset)
	if reSeOk {
		seReset.Reset()
	}
	seCloser, seOk := se.(io.Closer)
	if seOk {
		defer seCloser.Close()
	}
	cmd.Stderr = se

	err = cmd.Start()
	if err != nil {
		return cmd.ProcessState.ExitCode(), err
	}
	err = cmd.Wait()
	if err != nil {
		return cmd.ProcessState.ExitCode(), err
	}
	if sa.delay > 0.0 {
		time.Sleep(time.Duration(sa.delay) * time.Millisecond)
	}
	cp, ok := so.(ClipContent)
	if ok {
		if cp.ShouldClip() {
			mainWindow.Clipboard().SetContent(cp.GetContent())
		}
	}
	if outEncKey != "" {
		soE, ok := so.(Encrypted)
		if ok {
			soE.WriteToEncryptedFile(outEncKey)
		}
	}
	httpPost, ok := so.(*HttpPostWriter)
	if ok {
		err := httpPost.Post()
		if err != nil {
			return RC_ERROR, err
		}
	}

	return RC_CLEAN, nil
}

func substituteValuesIntoArgs(s []string, entryDialog func(*LocalValue) error, dataCache *DataCache) ([]string, error) {
	resp := make([]string, 0)
	for _, v := range s {
		tmp, err := substituteValuesIntoString(v, entryDialog, dataCache)
		if err != nil {
			return nil, err
		}
		resp = append(resp, tmp)
	}
	return resp, nil
}

func substituteValuesIntoString(s string, entryDialog func(*LocalValue) error, dataCache *DataCache) (string, error) {
	return dataCache.Template(s, entryDialog)
}

func derivePasswordFromName(name string, sa *SingleAction, dataCache *DataCache) (string, error) {
	if name != "" {
		lv, ok := dataCache.GetLocalValue(name)
		if ok {
			if lv.inputRequired && !lv.inputDone {
				err := validatedEntryDialog(lv)
				if err != nil {
					return "", err
				}
			}
			if lv.GetValue() == "" {
				return "", fmt.Errorf("password not provided")
			}
			return lv.GetValue(), nil
		}
	}
	return "", nil
}
