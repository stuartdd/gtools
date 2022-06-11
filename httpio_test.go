package main

import (
	"testing"
)

const (
	message = "Hello World"
)

func TestGet(t *testing.T) {
	rc, err := HttpPost("http://192.168.1.243:8080/files/user/stuart/loc/mydb/name/xxx.data", "text/plain", message)
	if err != nil {
		t.Fatalf("Post Response err:%s", err.Error())
	}
	if rc != 201 {
		t.Fatalf("Post Response not 201. actual:%d", rc)
	}

	resp, err := HttpGet("http://192.168.1.243:8080/files/user/stuart/loc/mydb/name/xxx.data")
	if err != nil {
		t.Fatalf("Get Response err:%s", err.Error())
	}
	if resp != message {
		t.Fatalf("Get Response = '%s'. actual:'%s'", message, resp)
	}
}
