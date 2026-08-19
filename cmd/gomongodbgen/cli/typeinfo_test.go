package cli

import (
	"github.com/xpwu/go-mongodb/field"
	"strings"
	"testing"
)

func TestNewTypeInfo_Valid(t *testing.T) {
	info := NewTypeInfo[int](newTestField)
	if info.Err != nil {
		t.Fatalf("unexpected error: %v", info.Err)
	}
	if info.T.Name() != "int" || info.T.PkgPath() != "" {
		t.Errorf("T = %s, %s", info.T.Name(), info.T.PkgPath())
	}
	if info.NewField.Name() != "newTestField" {
		t.Errorf("NewField = %q, want newTestField", info.NewField.Name())
	}
}

func TestNewTypeInfo_AnonymousFunction(t *testing.T) {
	info := NewTypeInfo[int](func(name string) field.Field { return nil })
	if info.Err == nil {
		t.Error("expected error for anonymous function, got nil")
	}
	if !strings.Contains(info.Err.Error(), "anonymous") {
		t.Errorf("err = %q, should mention anonymous", info.Err.Error())
	}
}

type methodValueReceiver struct{}

func (m *methodValueReceiver) makeField(name string) field.Field { return nil }

func TestNewTypeInfo_MethodValue(t *testing.T) {
	r := &methodValueReceiver{}
	info := NewTypeInfo[int](r.makeField)
	if info.Err == nil {
		t.Error("expected error for method value, got nil")
	}
	if !strings.Contains(info.Err.Error(), "method values") {
		t.Errorf("err = %q, should mention method values", info.Err.Error())
	}
}
