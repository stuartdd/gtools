package main

import (
	"fmt"
	"io"
	"os/exec"
	"time"
)

const (
	RC_FAIL  = -2
	RC_SETUP = -1
	RC_OK    = 0
)

func execDelayedAction(action *MultipleActionData, delay int, notifyChannel chan *NotifyMessage, dataCache *DataCache) {
	if delay == 0 {
		delay = 100
	}
	if notifyChannel != nil {
		notifyChannel <- NewNotifyMessage(LOG, action, "RunAtStart", fmt.Sprintf("delay = %d ms", delay), RC_OK, nil)
	}
	go func() {
		time.Sleep(time.Duration(delay) * time.Millisecond)
		execMultipleAction(action, notifyChannel, dataCache)
	}()
}

func execMultipleAction(data *MultipleActionData, notifyChannel chan *NotifyMessage, dataCache *DataCache) {
	if notifyChannel != nil {
		notifyChannel <- NewNotifyMessage(START, data, "Action Started", "", RC_OK, nil)
	}
	defer func() {
		if notifyChannel != nil {
			notifyChannel <- NewNotifyMessage(DONE, data, "Action Complete", "", RC_OK, nil)
		}
	}()
	stdOut := NewSysoutWriter("", stdColourPrefix[STD_OUT])
	stdErr := NewSysoutWriter("", stdColourPrefix[STD_ERR])
	for i, act := range data.commands {
		locationMsg := fmt.Sprintf("Action '%s' step '%d' path '%s'", data.desc, i, act.Dir())
		rc, err := execSingleAction(act, stdOut, stdErr, data.desc, dataCache)
		if err != nil {
			if rc == RC_SETUP {
				if notifyChannel != nil {
					notifyChannel <- NewNotifyMessage(ERROR, data, "Error Setup Process", locationMsg, rc, err)
				}
				return
			}
			if rc == RC_FAIL {
				if notifyChannel != nil {
					notifyChannel <- NewNotifyMessage(ERROR, data, "Failed Post Process", locationMsg, rc, err)
				}
				return
			}
			if act.ignoreError {
				if notifyChannel != nil {
					notifyChannel <- NewNotifyMessage(WARN, data, locationMsg, "", rc, err)
				}
			} else {
				exitOsMsg := fmt.Sprintf("Exit to OS with RC=%d", rc)
				if notifyChannel != nil {
					notifyChannel <- NewNotifyMessage(CMD_RC, data, exitOsMsg, "", rc, err)
				}
				return
			}
		}
	}
}

func execSingleAction(sa *SingleAction, stdOut, stdErr *SysoutWriter, actionDesc string, dataCache *DataCache) (int, error) {
	outEncKey, err := derivePasswordFromName(sa.outPwName, sa, dataCache)
	if err != nil {
		return RC_SETUP, err
	}
	inEncKey, err := derivePasswordFromName(sa.inPwName, sa, dataCache)
	if err != nil {
		return RC_SETUP, err
	}
	args, err := substituteValuesIntoArgs(sa.args, ValidatedEntryDialog, dataCache)
	if err != nil {
		return RC_SETUP, err
	}
	cmd := exec.Command(sa.command, args...)
	if sa.directory != "" {
		cmd.Dir = sa.directory
	}
	if sa.sysinDef != "" {
		tmp, err := substituteValuesIntoString(sa.sysinDef, SysInDialog, dataCache)
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
	sysoutDef, err := substituteValuesIntoString(sa.sysoutDef, SysOutDialog, dataCache)
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

	syserrDef, err := substituteValuesIntoString(sa.syserrDef, SysOutDialog, dataCache)
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
	//
	// Ready to exec the commands
	//
	err = cmd.Start()
	if err != nil {
		return cmd.ProcessState.ExitCode(), err
	}
	err = cmd.Wait()
	if err != nil {
		return cmd.ProcessState.ExitCode(), err
	}
	//
	// All writers and readers are complete!
	//
	if sa.delay > 0.0 {
		time.Sleep(time.Duration(sa.delay) * time.Millisecond)
	}

	cw, ok := so.(*CacheWriter)
	if ok {
		if cw.cacheType == MEM_TYPE {
			if notifyChannel != nil {
				notifyChannel <- NewNotifyMessage(SET_MEM, nil, fmt.Sprintf("%s=%s", cw.name, cw.GetContent()), "", 0, nil)
			}
		}
	}

	cp, ok := so.(ClipContent)
	if ok {
		if cp.ShouldClip() {
			if notifyChannel != nil {
				notifyChannel <- NewNotifyMessage(TO_CLIP, nil, fmt.Sprintf("Copied to Clipboard cmd:%s", sa.String()), "", 0, nil)
			}
			mainWindow.Clipboard().SetContent(cp.GetContent())
		}
	}

	if outEncKey != "" {
		soE, ok := so.(Encrypted)
		if ok {
			if notifyChannel != nil {
				notifyChannel <- NewNotifyMessage(SAVE_EN, nil, fmt.Sprintf("Save to enc file:%s", cw.name), "", 0, nil)
			}
			err = soE.SaveToEncryptedFile(outEncKey)
			if err != nil {
				return RC_FAIL, err
			}
		}
	}

	httpPost, ok := so.(*HttpPostWriter)
	if ok {
		err := httpPost.Post()
		if err != nil {
			return RC_FAIL, err
		}
	}
	return RC_OK, nil
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
				err := ValidatedEntryDialog(lv)
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
