/*
* bridge.go
* GoHue library for Philips Hue
* Copyright (C) 2016 Collin Guarino (Collinux) collinux[-at-]users.noreply.github.com
* License: GPL version 2 or higher http://www.gnu.org/licenses/gpl.html
 */
// All things start with the bridge. You will find many Bridge.Func() items
// to use once a bridge has been created and identified.
// See the getting started guide on the Philips hue website:
// http://www.developers.meethue.com/documentation/getting-started

package hue

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Bridge struct defines hardware that is used to communicate with the lights.
type Bridge struct {
	IPAddress string `json:"internalipaddress"`
	Username  string // Token from Bridge.CreateUser
	Info      BridgeInfo
}

// BridgeInfo struct is the format for parsing xml from a bridge.
type BridgeInfo struct {
	XMLName xml.Name `xml:"root"`
	Device  struct {
		XMLName          xml.Name `xml:"device"`
		DeviceType       string   `xml:"deviceType"`
		FriendlyName     string   `xml:"friendlyName"`
		Manufacturer     string   `xml:"manufacturer"`
		ManufacturerURL  string   `xml:"manufacturerURL"`
		ModelDescription string   `xml:"modelDescription"`
		ModelName        string   `xml:"modelName"`
		ModelNumber      string   `xml:"modelNumber"`
		ModelURL         string   `xml:"modelURL"`
		SerialNumber     string   `xml:"serialNumber"`
		UDN              string   `xml:"UDN"`
	} `xml:"device"`
}

func (b *Bridge) newClient() *http.Client {
	return &http.Client{Timeout: time.Second * 5}
}

func (b *Bridge) uri(path string) string {
	return fmt.Sprintf("http://%s%s", b.IPAddress, path)
}

// Get sends a http GET to the bridge
func (bridge *Bridge) Get(path string) ([]byte, io.Reader, error) {
	uri := bridge.uri(path)
	log.Println("GET:", uri)
	client := bridge.newClient()
	resp, err := client.Get(uri)
	if err != nil {
		return []byte{}, nil, fmt.Errorf("unable to access bridge: %w", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, nil, fmt.Errorf("unable to read response body: %w", err)
	}
	return body, bytes.NewReader(body), nil
}

// Put sends a http PUT to the bridge with
// a body formatted with parameters (in a generic interface)
func (bridge *Bridge) Put(path string, params interface{}) ([]byte, io.Reader, error) {
	uri := bridge.uri(path)
	log.Println("PUT:", uri)
	data, err := json.Marshal(params)
	if err != nil {
		return []byte{}, nil, fmt.Errorf("unable to marshal PUT request interface: %w", err)
	}

	request, err := http.NewRequest("PUT", uri, bytes.NewReader(data))
	if err != nil {
		return []byte{}, nil, fmt.Errorf("unable to create PUT request: %w", err)
	}

	client := bridge.newClient()
	resp, err := client.Do(request)
	if err != nil {
		return []byte{}, nil, fmt.Errorf("unable to access bridge: %w", err)
	}
	return HandleResponse(resp)
}

// Post sends a http POST to the bridge with
// a body formatted with parameters (in a generic interface).
// If `params` is nil then it will send an empty body with the post request.
func (bridge *Bridge) Post(path string, params interface{}) ([]byte, io.Reader, error) {
	// Add the params to the request or allow an empty body
	var request []byte
	if params != nil {
		reqBody, err := json.Marshal(params)
		if err != nil {
			return []byte{}, nil, fmt.Errorf("unable to add POST body parameters due to json marshalling error: %w", err)
		}
		request = reqBody
	}

	// Send the request and handle the response
	uri := bridge.uri(path)
	log.Println("POST:", uri)
	client := bridge.newClient()
	resp, err := client.Post(uri, "text/json", bytes.NewReader(request))
	if err != nil {
		return []byte{}, nil, fmt.Errorf("unable to access bridge: %w", err)
	}

	return HandleResponse(resp)
}

// Delete sends a http DELETE to the bridge
func (bridge *Bridge) Delete(path string) error {
	uri := bridge.uri(path)
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		return fmt.Errorf("unable to create DELETE request: %w", err)
	}

	client := bridge.newClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to access bridge: %w", err)
	}

	_, _, err = HandleResponse(resp)
	if err != nil {
		return fmt.Errorf("unable to access bridge: %w", err)
	}

	return nil
}

// HandleResponse manages the http.Response content from a
// bridge Get/Put/Post/Delete by checking it for errors
// and invalid return types.
func HandleResponse(resp *http.Response) ([]byte, io.Reader, error) {
	log.Printf("code: %d", resp.StatusCode)
	log.Printf("headers: %+v", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, nil, fmt.Errorf("unable to read response: %w", err)
	}
	defer resp.Body.Close()

	reader := bytes.NewReader(body)
	if strings.Contains(string(body), "\"error\"") {
		errString := string(body)
		errNum := errString[strings.Index(errString, "type\":")+6 : strings.Index(errString, ",\"address")]
		errDesc := errString[strings.Index(errString, "description\":\"")+14 : strings.Index(errString, "\"}}")]
		return []byte{}, nil, fmt.Errorf("failed to handle response: error type %s: %s", errNum, errDesc)
	}

	return body, reader, nil
}

// FindBridges will visit www.meethue.com/api/nupnp to see a list of
// bridges on the local network.
func FindBridges() ([]Bridge, error) {
	bridge := Bridge{IPAddress: "www.meethue.com"}
	body, _, err := bridge.Get("/api/nupnp")
	if err != nil {
		return []Bridge{}, fmt.Errorf("unable to locate bridge: %w", err)
	}

	var bridges []Bridge
	err = json.Unmarshal(body, &bridges)
	if err != nil {
		return bridges, fmt.Errorf("unable to unmarshal bridge list: %w", err)
	}
	if len(bridges) == 0 {
		return bridges, errors.New("no bridges found")
	}
	return bridges, nil
}

// NewBridge defines hardware that is compatible with Hue.
// The function is the core of all functionality, it's necessary
// to call `NewBridge` and `Login` or `CreateUser` to access any
// lights, scenes, groups, etc.
func NewBridge(ip string) (*Bridge, error) {
	bridge := Bridge{
		IPAddress: ip,
	}
	// Test the connection by attempting to get the bridge info.
	err := bridge.GetInfo()
	if err != nil {
		return &Bridge{}, err
	}
	return &bridge, nil
}

// GetInfo retrieves the description.xml file from the bridge.
// This is used as a check to see if the bridge is accessible
// and any error will be fatal as the bridge is required for nearly
// all functions.
func (bridge *Bridge) GetInfo() error {
	_, reader, err := bridge.Get("/description.xml")
	if err != nil {
		return err
	}
	data := BridgeInfo{}
	err = xml.NewDecoder(reader).Decode(&data)
	if err != nil {
		return fmt.Errorf("failed to decode xml response from bridge description: %w", err)
	}

	bridge.Info = data
	log.Printf("Connected to bridge: %+v\n", bridge.Info)

	return nil
}

// Login verifies that the username token has bridge access
// and only assigns the bridge its Username value if verification is successful.
func (bridge *Bridge) Login(username string) error {
	uri := fmt.Sprintf("/api/%s", username)
	_, _, err := bridge.Get(uri)
	if err != nil {
		return err
	}
	bridge.Username = username
	return nil
}

// CreateUser adds a new user token on the whitelist.
// and returns this value as a string.
//
// The 'Bridge.Login` function **must be run** with
// the user token as an argument. No functions can
// be called until a valid user token is assigned as the
// bridge's `Username` value.
//
// You cannot set a plaintext username, it must be a
// generated user token. This was done by Philips Hue for security purposes.
func (bridge *Bridge) CreateUser(deviceType string) (string, error) {
	params := map[string]string{"devicetype": deviceType}
	body, _, err := bridge.Post("/api", params)
	if err != nil {
		return "", err
	}
	content := string(body)
	username := content[strings.LastIndex(content, ":\"")+2 : strings.LastIndex(content, "\"")]
	bridge.Username = username
	return username, nil
}

// DeleteUser deletes a user given its USER KEY, not the string name.
// See http://www.developers.meethue.com/documentation/configuration-api
// for description on `username` deprecation in place of the devicetype key.
func (bridge *Bridge) DeleteUser(username string) error {
	uri := fmt.Sprintf("/api/%s/config/whitelist/%s", bridge.Username, username)
	err := bridge.Delete(uri)
	if err != nil {
		return err
	}
	return nil
}

// GetAllLights retrieves the state of all lights that the bridge is aware of.
func (bridge *Bridge) GetAllLights() ([]Light, error) {
	uri := fmt.Sprintf("/api/%s/lights", bridge.Username)
	body, _, err := bridge.Get(uri)
	if err != nil {
		return []Light{}, err
	}

	// An index is at the top of every Light in the array
	lightMap := map[string]Light{}
	err = json.Unmarshal(body, &lightMap)
	if err != nil {
		return []Light{}, fmt.Errorf("unable to marshal GetAllLights response: %w", err)
	}

	// Parse the index, add the light to the list, and return the array
	var lights []Light
	for index, light := range lightMap {
		light.Index, err = strconv.Atoi(index)
		if err != nil {
			return []Light{}, fmt.Errorf("unable to convert light index to integer: %w", err)
		}
		light.Bridge = bridge
		lights = append(lights, light)
	}
	return lights, nil
}

// GetLightByIndex returns a light struct containing data on
// a light given its index stored on the bridge. This is used for
// quickly updating an individual light.
func (bridge *Bridge) GetLightByIndex(index int) (Light, error) {
	// Send a http GET and inspect the response
	uri := fmt.Sprintf("/api/%s/lights/%d", bridge.Username, index)
	body, _, err := bridge.Get(uri)
	if err != nil {
		return Light{}, err
	}
	if strings.Contains(string(body), "not available") {
		return Light{}, errors.New("Error: Light selection index out of bounds. ")
	}

	// Parse and load the response into the light array
	light := Light{}
	err = json.Unmarshal(body, &light)
	if err != nil {
		return Light{}, fmt.Errorf("unable to unmarshal light data: %w", err)
	}
	light.Index = index
	light.Bridge = bridge
	return light, nil
}

// FindNewLights makes the bridge search the zigbee spectrum for
// lights in the area and will add them to the list of lights available.
// If successful these new lights can be used by `Bridge.GetAllLights`
//
// Notes from Philips Hue API documentation:
// The bridge will search for 1 minute and will add a maximum of 15 new
// lights. To add further lights, the command needs to be sent again after
// the search has completed. If a search is already active, it will be
// aborted and a new search will start.
// http://www.developers.meethue.com/documentation/lights-api#13_search_for_new_lights
func (bridge *Bridge) FindNewLights() error {
	uri := fmt.Sprintf("/api/%s/lights", bridge.Username)
	_, _, err := bridge.Post(uri, nil)
	if err != nil {
		return err
	}
	return nil
}

// GetLightByName returns a light struct containing data on a given name.
func (bridge *Bridge) GetLightByName(name string) (Light, error) {
	lights, _ := bridge.GetAllLights()
	for _, light := range lights {
		if light.Name == name {
			return light, nil
		}
	}
	return Light{}, fmt.Errorf("light named '%s' not found", name)
}

// GetAllSensors retrieves the state of all sensors that the bridge is aware of.
func (bridge *Bridge) GetAllSensors() ([]Sensor, error) {
	uri := fmt.Sprintf("/api/%s/sensors", bridge.Username)
	body, _, err := bridge.Get(uri)
	if err != nil {
		return []Sensor{}, err
	}

	// An index is at the top of every sensor in the array
	sensorList := map[string]Sensor{}
	err = json.Unmarshal(body, &sensorList)
	if err != nil {
		return []Sensor{}, fmt.Errorf("unable to marshal GetAllSensors response: %w", err)
	}

	// Parse the index, add the sensor to the list, and return the array
	sensors := make([]Sensor, 0, len(sensorList))
	for index, sensor := range sensorList {
		sensor.Index, err = strconv.Atoi(index)
		if err != nil {
			return []Sensor{}, fmt.Errorf("unable to convert sensor index to integer: %w", err)
		}
		sensor.Bridge = bridge
		sensors = append(sensors, sensor)
	}
	return sensors, nil
}

// GetSensorByIndex returns a sensor struct containing data on
// a sensor given its index stored on the bridge.
func (bridge *Bridge) GetSensorByIndex(index int) (Sensor, error) {
	// Send a http GET and inspect the response
	uri := fmt.Sprintf("/api/%s/sensors/%d", bridge.Username, index)
	body, _, err := bridge.Get(uri)
	if err != nil {
		return Sensor{}, err
	}
	if strings.Contains(string(body), "not available") {
		return Sensor{}, errors.New("Sensor selection index out of bounds. ")
	}

	// Parse and load the response into the sensor array
	sensor := Sensor{}
	err = json.Unmarshal(body, &sensor)
	if err != nil {
		return Sensor{}, fmt.Errorf("unable to unmarshal light data: %w", err)
	}
	sensor.Index = index
	sensor.Bridge = bridge
	return sensor, nil
}
