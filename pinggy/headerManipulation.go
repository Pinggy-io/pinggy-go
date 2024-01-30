package pinggy

import (
	"encoding/base64"
	"fmt"
	"strings"
)

/*
New HeaderManipulationAndConfig struct. This is a safe way create the object
*/
func NewHeaderManipulationAndAuthConfig() *HttpHeaderManipulationAndAuthConfig {
	return &HttpHeaderManipulationAndAuthConfig{
		Headers:     map[string]*PinggyHttpHeaderInfo{},
		BasicAuths:  make(map[string]bool),
		BearerAuths: make(map[string]bool),
	}
}

/*
Add username password, basic authentication.
One can add more than one username password.
Along with bearer auth.
*/
func (hmd *HttpHeaderManipulationAndAuthConfig) AddBasicAuth(username, password string) {
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	hmd.BasicAuths[encodedAuth] = true
}

/*
Add bearer key authentication. It would enable the bearer authentication mode.
It can be use along with username password authentication.
*/
func (hmd *HttpHeaderManipulationAndAuthConfig) AddBearerAuth(key string) {
	hmd.BearerAuths[key] = true
}

/*
Set hostname.
*/
func (hmd *HttpHeaderManipulationAndAuthConfig) SetHostname(hostname string) {
	hmd.HostName = strings.ToLower(hostname)
}

func (hmd *HttpHeaderManipulationAndAuthConfig) getValue(headerName string) (*PinggyHttpHeaderInfo, error) {
	headerLower := strings.ToLower(headerName)
	if headerLower == "host" {
		return nil, fmt.Errorf("to set host header use `SetHostname`")
	}
	value, ok := hmd.Headers[headerLower]
	if !ok {
		value = &PinggyHttpHeaderInfo{Key: headerName, NewValues: make([]string, 0)}
		hmd.Headers[headerLower] = value
	}
	return value, nil
}

/*
Remove a particular header from the http request.
*/
func (hmd *HttpHeaderManipulationAndAuthConfig) RemoveHeader(headerName string) error {
	value, err := hmd.getValue(headerName)
	if err != nil {
		return err
	}
	value.Remove = true
	return nil
}

/*
Add header to the http request. It would not remove same existing header.
*/
func (hmd *HttpHeaderManipulationAndAuthConfig) AddHeader(headerName, headerValue string) error {
	value, err := hmd.getValue(headerName)
	if err != nil {
		return err
	}
	value.NewValues = append(value.NewValues, headerValue)
	return nil
}

/*
Update a header. It would remove existing header and append new headers. It would append new headers
even if same header does not exists.
*/
func (hmd *HttpHeaderManipulationAndAuthConfig) UpdateHeader(headerName, headerValue string) error {
	value, err := hmd.getValue(headerName)
	if err != nil {
		return err
	}
	value.NewValues = append(value.NewValues, headerValue)
	value.Remove = true
	return nil
}
