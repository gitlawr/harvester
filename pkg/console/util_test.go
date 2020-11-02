package console

import "testing"

func TestGetHarvesterManifestContent(t *testing.T) {
	d := map[string]string{
		"a": "b",
		"b": "\"c\"",
	}
	res := getHarvesterManifestContent(d)
	t.Log(res)
}

func TestGetHStatus(t *testing.T) {
	s := getHarvesterStatus()
	t.Log(s)
}

func TestFoo(t *testing.T) {
	var m map[string]string
	var l interface{} = &m
	temp, ok := l.(*map[string]string)
	if !ok {
		t.Error("not ok")
	}
	t.Log(temp)
}
