package headermanipulation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type PinggyHttpHeaderInfo struct {
	Key       string   `json:"headerName"`
	Remove    bool     `json:"remove"`
	NewValues []string `json:"values"`
}

type HttpHeaderManipulationAndAuthConfig struct {
	HostName       string                           `json:"hostName"`
	Headers        map[string]*PinggyHttpHeaderInfo `json:"headers"`
	BasicAuths     map[string]bool                  `json:"basicAuths"`
	BearerAuths    map[string]bool                  `json:"bearerAuths"`
	XFF            string                           `json:"xff"`            //header name. empty means not do not set
	HttpsOnly      bool                             `json:"httpsOnly"`      //All the http would be redirected
	FullRequestUrl bool                             `json:"fullRequestUrl"` //Will add X-Pinggy-Url to add entire url
	PassPreflight  bool                             `json:"allowPreflight"` //Allow CORS Preflight URL through auth
	XFH            bool                             `json:"xfh"`
	XFP            bool                             `json:"xfp"`
	Forwarded      bool                             `json:"forwarded"`
}

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

/*
Get hostname.
*/
func (hmd *HttpHeaderManipulationAndAuthConfig) GetHostname() string {
	return hmd.HostName
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

func (hmd *HttpHeaderManipulationAndAuthConfig) ListHeaderManipulations() ([]byte, error) {
	values := hmd
	jb, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return jb, nil
}

func (hmd *HttpHeaderManipulationAndAuthConfig) ReconstructHeaderManipulationDataFromJson(jsondata []byte) error {
	newHmd := HttpHeaderManipulationAndAuthConfig{}
	err := json.Unmarshal(jsondata, &newHmd)
	if err != nil {
		return err
	}

	hmd.HostName = newHmd.HostName
	hmd.Headers = make(map[string]*PinggyHttpHeaderInfo)
	for key, hi := range newHmd.Headers {
		headerLower := strings.ToLower(key)
		if headerLower == "host" {
			continue
		}
		hmd.Headers[headerLower] = hi
	}
	hmd.BasicAuths = newHmd.BasicAuths
	hmd.BearerAuths = newHmd.BearerAuths
	hmd.FullRequestUrl = newHmd.FullRequestUrl
	hmd.XFF = newHmd.XFF
	hmd.HttpsOnly = newHmd.HttpsOnly
	hmd.PassPreflight = newHmd.PassPreflight
	hmd.XFH = newHmd.XFH
	hmd.XFP = newHmd.XFP
	hmd.Forwarded = newHmd.Forwarded
	return nil
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetXFFHeader(xff string) {
	hmd.XFF = xff
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetXFF() {
	hmd.XFF = "X-Forwarded-For"
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetHttpsOnly(val bool) {
	hmd.HttpsOnly = val
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetFullUrl(val bool) {
	hmd.FullRequestUrl = val
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetPassPreflight(val bool) {
	hmd.PassPreflight = val
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetXFH(val bool) {
	hmd.XFH = val
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetXFP(val bool) {
	hmd.XFP = val
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetForwarded(val bool) {
	hmd.Forwarded = val
}

func (hmd *HttpHeaderManipulationAndAuthConfig) SetReverseProxy(hostname string) {
	hmd.SetXFF()
	hmd.SetXFH(true)
	hmd.SetXFP(true)
	hmd.SetForwarded(true)
	hmd.SetHostname(hostname)
}
