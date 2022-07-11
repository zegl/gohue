/*
* bridge_test.go
* GoHue library for Philips Hue
* Copyright (C) 2016 Collin Guarino (Collinux) collinux[-at-]users.noreply.github.com
* License: GPL version 2 or higher http://www.gnu.org/licenses/gpl.html
 */

package hue

import (
	"encoding/xml"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func TestCreateUser(t *testing.T) {
	bridges, err := FindBridges()
	if err != nil {
		t.Fatal(err)
	}
	bridge := bridges[0]
	username, _ := bridge.CreateUser("test")
	bridge.Login(username)
	//bridge.DeleteUser(bridge.Username)
}

func TestFindBridges(t *testing.T) {
	bridges, err := FindBridges()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(bridges)
}

func TestBridgeLogin(t *testing.T) {
	bridges, err := FindBridges()
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("HUE_USER_TOKEN") == "" {
		t.Fatal("The environment variable HUE_USER_TOKEN must be set to the value from bridge.CreateUser")
	}
	bridges[0].Login(os.Getenv("HUE_USER_TOKEN"))
}

func Test2(t *testing.T) {
	reader := strings.NewReader(`
<?xml version="1.0" encoding="UTF-8" ?>
<root xmlns="urn:schemas-upnp-org:device-1-0">
<specVersion>
<major>1</major>
<minor>0</minor>
</specVersion>
<URLBase>http://192.168.86.27:80/</URLBase>
<device>
<deviceType>urn:schemas-upnp-org:device:Basic:1</deviceType>
<friendlyName>Philips hue (192.168.86.27)</friendlyName>
<manufacturer>Signify</manufacturer>
<manufacturerURL>http://www.philips-hue.com</manufacturerURL>
<modelDescription>Philips hue Personal Wireless Lighting</modelDescription>
<modelName>Philips hue bridge 2015</modelName>
<modelNumber>BSB002</modelNumber>
<modelURL>http://www.philips-hue.com</modelURL>
<serialNumber>ecb5fa2a484e</serialNumber>
<UDN>uuid:2f402f80-da50-11e1-9b23-ecb5fa2a484e</UDN>
<presentationURL>index.html</presentationURL>
<iconList>
<icon>
<mimetype>image/png</mimetype>
<height>48</height>
<width>48</width>
<depth>24</depth>
<url>hue_logo_0.png</url>
</icon>
</iconList>
</device>
</root>
`)

	data := BridgeInfo{}
	err := xml.NewDecoder(reader).Decode(&data)
	assert.NoError(t, err)
}

func TestNewBridge(t *testing.T) {
	bridge, err := NewBridge("192.168.86.27")
	assert.NoError(t, err)
	assert.NotNil(t, bridge)

	err = bridge.Login("O-PpzqSJZW2a3aTiQ6lVayMVkkQ-HapLY0qinC3x")
	assert.NoError(t, err)
}
