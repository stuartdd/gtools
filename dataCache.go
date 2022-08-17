package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

type LocalValue struct {
	name          string
	desc          string
	_value        string
	lastValue     string // Use by FileSave and FileOpen as that last location used
	minLen        int
	isPassword    bool
	isFileName    bool
	isFileWatch   bool
	inputDone     bool
	inputRequired bool
	notifyChannel chan *NotifyMessage
}

type DataCache struct {
	mu          sync.Mutex
	memoryMap   map[string]*CacheWriter
	localVarMap map[string]*LocalValue
	envMap      map[string]string
}

func newLocalValue(name, desc, defaultVal string, minLen int, isPassword, isFileName, isFileWatch, inputRequired bool) *LocalValue {
	return &LocalValue{name: name, desc: desc, _value: defaultVal, minLen: minLen, lastValue: "", isPassword: isPassword, isFileName: isFileName, isFileWatch: isFileWatch, inputDone: false, inputRequired: inputRequired}
}

func (lv *LocalValue) String() string {
	if lv.isPassword {
		return fmt.Sprintf("LocalValue name:%s minLen:%d, value:\"?\", isPW:%t, isFN:%t, isFW:%t", lv.name, lv.minLen, lv.isPassword, lv.isFileName, lv.isFileWatch)
	}
	return fmt.Sprintf("LocalValue name:%s minLen:%d, value:\"%s\", isPW:%t, isFN:%t, isFW:%t", lv.name, lv.minLen, lv._value, lv.isPassword, lv.isFileName, lv.isFileWatch)
}

func (v *LocalValue) GetValue() string {
	if v.isFileWatch {
		f, err := os.Open(v._value)
		if err != nil {
			return fmt.Sprintf("%%{%s}", v.name)
		}
		defer f.Close()
	}
	return v._value
}

func (v *LocalValue) SetValue(val string) {
	if v.notifyChannel != nil {
		if v.isPassword {
			v.notifyChannel <- NewNotifyMessage(SET_LOC, nil, fmt.Sprintf("Set Local Password: %s", v.name), "", 0, nil)
		} else {
			v.notifyChannel <- NewNotifyMessage(SET_LOC, nil, fmt.Sprintf("Set Local Value: %s=%s", v.name, val), "", 0, nil)
		}
	}
	v._value = val
}

func (v *LocalValue) GetLastValueAsLocation() (fyne.ListableURI, error) {
	if v.lastValue == "" {
		d, err := os.Getwd()
		if err == nil {
			v.lastValue = d
		}
	}
	u, err := storage.ParseURI("file://" + v.lastValue)
	if err != nil {
		return nil, err
	}
	l, err := storage.ListerForURI(u)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func NewDataCache() *DataCache {
	m := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			m[pair[0]] = pair[1]
		}
	}
	return &DataCache{memoryMap: make(map[string]*CacheWriter), localVarMap: make(map[string]*LocalValue), envMap: m}
}

func (dc *DataCache) LogLocalValues(debugLog *LogData) {
	if debugLog.IsLogging() {
		for _, lv := range dc.localVarMap {
			debugLog.WriteLog(lv.String())
		}
	}
}

func (dc *DataCache) GetLocalValue(name string) (*LocalValue, bool) {
	lv, found := dc.localVarMap[name]
	return lv, found
}

func (dc *DataCache) MergeLocalValuesMap(localMod *DataCache) {
	for n, v := range localMod.localVarMap {
		dc.localVarMap[n] = v
	}
}

func (dc *DataCache) AddLocalValue(name, desc, defaultVal string, minLen int, isPassword, isFileName, isFileWatch, inputRequired bool) *LocalValue {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	lv := newLocalValue(name, desc, defaultVal, minLen, isPassword, isFileName, isFileWatch, inputRequired)
	dc.localVarMap[name] = lv
	return lv
}

func (dc *DataCache) ResetCache() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.memoryMap = make(map[string]*CacheWriter)
}

func (dc *DataCache) PutCacheWriter(cw *CacheWriter) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.memoryMap[cw.name] = cw
}

func (dc *DataCache) GetCacheWriter(name string) *CacheWriter {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	c, ok := dc.memoryMap[name]
	if ok {
		return c
	}
	return nil
}

func (dc *DataCache) Template(s string, dialogFunc func(*LocalValue) error) (string, error) {
	return TemplateParse(s, func(name string) (string, error) {
		if name == "" {
			return "%{}", nil
		}
		cw := dc.GetCacheWriter(name)
		if cw != nil {
			return cw.GetContent(), nil
		}
		lv, found := dc.localVarMap[name]
		if found {
			if !lv.inputDone && lv.inputRequired && dialogFunc != nil {
				err := dialogFunc(lv)
				if err != nil {
					return "", err
				}
			}
			return lv.GetValue(), nil
		}
		s, found := dc.envMap[name]
		if found {
			return s, nil
		}
		return fmt.Sprintf("%%{%s}", name), nil
	})
}
