/*
* scene_test.go
* GoHue library for Philips Hue
* Copyright (C) 2016 Collin Guarino (Collinux) collinux[-at-]users.noreply.github.com
* License: GPL version 2 or higher http://www.gnu.org/licenses/gpl.html
 */

package hue

import (
	"testing"
	"os"
)

func TestGetAllScenes(t *testing.T) {
	bridges, err := FindBridges()
	if err != nil {
		t.Fatal(err)
	}
	bridge := bridges[0]
	if os.Getenv("HUE_USER_TOKEN") == "" {
		t.Fatal("The environment variable HUE_USER_TOKEN must be set to the value from bridge.CreateUser")
	}
	bridge.Login(os.Getenv("HUE_USER_TOKEN"))
	scenes, err := bridge.GetAllScenes()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(scenes)
}

// TODO not functional
// func TestCreateScene(t *testing.T) {
// 	bridges, err := hue.FindBridges()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	bridge := bridges[0]
// 	bridge.Login(os.Getenv("HUE_USER_TOKEN"))
// 	scene := hue.Scene{Name: "Testing", Lights: []string{"1", "2"}}
// 	err = bridge.CreateScene(scene)
// 	if err != nil {
// t.Fatal(err)
// }
// }
