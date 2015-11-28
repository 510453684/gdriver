package gdriver

import (
	"fmt"
	"testing"
)

type mockDriver struct{}

func (m *mockDriver) Init() string { return "init" }

type tDriver1 struct{}

func (t *tDriver1) New() interface{} { return &mockDriver{} }
func (t *tDriver1) Identity(id int) string {
	switch id {
	case IDENT_NAME:
		return "name"
	case IDENT_SHORT:
		return "short"
	case IDENT_LONG:
		return "long"
	}
	return "unknown"
}

func TestRegister(t *testing.T) {
	fmt.Printf("")
	Register("group", &tDriver1{})
	lst := ListGroup()

	if len(lst) < 1 {
		t.Error("Len should be 1 is %d", len(lst))
	}
	if _, ok := lst["group"]; !ok {
		t.Error("Group does not contain our group name")
	}

	ndrive, err := New("group", "NAME")
	if err != nil {
		t.Error("Driver not registered %s", err.Error)
	}

	if !IsRegistered("group", "NAME") {
		t.Error("Couldn't find driver using IsRegistered()")
	}

	// We have to cast as a pointer because that is what is returned.
	initName := ndrive.(*mockDriver).Init()
	if initName != "init" {
		t.Error("Init did not get called")
	}

	if "name" != Help("group", "name", IDENT_NAME) {
		t.Error("Did not get 'name' for identity call")
	}

	if "short" != Help("group", "name", IDENT_SHORT) {
		t.Error("Did not get 'short' for identity call")
	}

	if "long" != Help("group", "name", IDENT_LONG) {
		t.Error("Did not get 'long' for identity call")
	}

	_, err = New("group", DEFAULT_DRIVER)
	if err != nil {
		t.Error("Should have had an error but didn't")
	}

	if !Default("group", "NAME") {
		t.Error(err.Error())
	}

	if defaultDriver, err := New("group", "name"); err != nil {
		t.Error(err.Error())
	} else {
		if "init" != defaultDriver.(*mockDriver).Init() {
			t.Error("Did not get init string from routine")
		}
	}

}
