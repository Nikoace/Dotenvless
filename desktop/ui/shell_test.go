package ui

import (
	"reflect"
	"testing"
)

func TestChineseManagementSections(t *testing.T) {
	if !reflect.DeepEqual(Sections(), []string{"工程", "共享库", "状态与接入"}) {
		t.Fatal("DESK-01: management navigation is missing")
	}
}
