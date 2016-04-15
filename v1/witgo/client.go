// Copyright 2016 Arne Roomann-Kurrik
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package witgo

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type HttpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

type LoggingHttpClient struct {
	log    io.Writer
	client HttpClient
}

func NewLoggingHttpClient(log io.Writer, client HttpClient) *LoggingHttpClient {
	return &LoggingHttpClient{
		log:    log,
		client: client,
	}
}

func (c *LoggingHttpClient) logReq(req *http.Request) *http.Request {
	var (
		buf *bytes.Buffer
		out *http.Request
		err error
	)
	io.WriteString(c.log, "=====\nHTTP Req\n-----\n")
	buf = bytes.NewBufferString("")
	req.Write(buf)
	if out, err = http.ReadRequest(bufio.NewReader(io.TeeReader(buf, c.log))); err != nil {
		panic(err)
	}
	// Make request usable as a client request.
	out.RequestURI = ""
	out.URL = req.URL
	return out
}

func (c *LoggingHttpClient) logResp(resp *http.Response) *http.Response {
	var (
		buf *bytes.Buffer
		out *http.Response
		err error
	)
	io.WriteString(c.log, "=====\nHTTP Resp\n-----\n")
	buf = bytes.NewBufferString("")
	resp.Write(buf)
	if out, err = http.ReadResponse(bufio.NewReader(io.TeeReader(buf, c.log)), nil); err != nil {
		panic(err)
	}
	io.WriteString(c.log, "\n")
	return out
}

func (c *LoggingHttpClient) Do(req *http.Request) (resp *http.Response, err error) {
	req = c.logReq(req)
	if resp, err = c.client.Do(req); resp != nil {
		resp = c.logResp(resp)
	}
	return
}

type Client struct {
	ServerAccessToken string
	HttpClient        HttpClient
	Version           string
	Base              string
	UserAgent         string
}

func NewClient(accessToken string) *Client {
	return &Client{
		ServerAccessToken: accessToken,
		HttpClient:        &http.Client{},
		Version:           "20160412",
		Base:              "https://api.wit.ai",
		UserAgent:         "github.com/kurrik/witgo",
	}
}

func (c *Client) buildRequest(
	method string,
	path string,
	query url.Values,
	contentType string,
	body io.Reader,
) (request *http.Request, err error) {
	var (
		authHeader string
		requestUrl string
	)
	if query == nil {
		query = url.Values{}
	}
	query.Set("v", c.Version)
	requestUrl = fmt.Sprintf("%v%v?%v", c.Base, path, query.Encode())
	if request, err = http.NewRequest(method, requestUrl, body); err != nil {
		return
	}
	request.Header.Set("Accept", "application/json")
	authHeader = fmt.Sprintf("Bearer %v", c.ServerAccessToken)
	request.Header.Set("Authorization", authHeader)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("User-Agent", c.UserAgent)
	return
}

func (c *Client) buildGetRequest(path string, fields map[string]string) (request *http.Request, err error) {
	var query = url.Values{}
	for field, value := range fields {
		if value != "" {
			query.Set(field, value)
		}
	}
	request, err = c.buildRequest("GET", path, query, "", nil)
	return
}

func (c *Client) buildPostRequest(path string, fields map[string]string, payload interface{}) (request *http.Request, err error) {
	var (
		query = url.Values{}
		body  io.ReadWriter
	)
	for field, value := range fields {
		if value != "" {
			query.Set(field, value)
		}
	}
	if payload != nil {
		if body, err = encodeJson(payload); err != nil {
			return
		}
	}
	request, err = c.buildRequest("POST", path, query, "application/json", body)
	return
}

func (c *Client) buildMultipartRequest(path string, fields map[string]string) (request *http.Request, err error) {
	var (
		typeHeader string
		body       io.ReadWriter
		encoder    *multipart.Writer
	)
	body = bytes.NewBufferString("")
	encoder = multipart.NewWriter(body)
	defer encoder.Close()
	for field, value := range fields {
		encoder.WriteField(field, value)
	}
	typeHeader = fmt.Sprintf("multipart/form-data;boundary=%v", encoder.Boundary())
	request, err = c.buildRequest("POST", path, nil, typeHeader, body)
	return
}

func (c *Client) makeRequest(request *http.Request) (response *Response, err error) {
	var r *http.Response
	r, err = c.HttpClient.Do(request)
	response = (*Response)(r)
	return
}

func (c *Client) Message(msg string) (response *Response, err error) {
	var request *http.Request
	if request, err = c.buildGetRequest("/message", map[string]string{
		"q": msg,
	}); err != nil {
		return
	}
	if response, err = c.makeRequest(request); err != nil {
		return
	}
	return
}

func (c *Client) Converse(sessionID SessionID, q string, context interface{}) (response *Response, err error) {
	var (
		request *http.Request
	)
	if request, err = c.buildPostRequest("/converse", map[string]string{
		"q":          q,
		"session_id": string(sessionID),
	}, context); err != nil {
		return
	}
	if response, err = c.makeRequest(request); err != nil {
		return
	}
	return
}

const (
	STATUS_OK = 200
)

// Error returned if there was an issue parsing the response body.
type ResponseError struct {
	Body string
	Code int
}

func NewResponseError(code int, body string) ResponseError {
	return ResponseError{Code: code, Body: body}
}

func (e ResponseError) Error() string {
	return fmt.Sprintf("Unable to handle response with code %d): `%v`", e.Code, e.Body)
}

type Response http.Response

func (r Response) readBody() (b []byte, err error) {
	var (
		header string
		reader io.Reader
	)
	defer r.Body.Close()
	header = strings.ToLower(r.Header.Get("Content-Encoding"))
	if header == "" || strings.Index(header, "gzip") == -1 {
		reader = r.Body
	} else {
		if reader, err = gzip.NewReader(r.Body); err != nil {
			return
		}
	}
	if b, err = ioutil.ReadAll(reader); err != nil {
		return
	}
	return
}

func (r Response) ReadBody() string {
	var (
		b   []byte
		err error
	)
	if b, err = r.readBody(); err != nil {
		return ""
	}
	return string(b)
}

// Parses a JSON encoded HTTP response into the supplied interface.
func (r Response) Parse(out interface{}) (err error) {
	var b []byte
	switch r.StatusCode {
	case STATUS_OK:
		if b, err = r.readBody(); err != nil {
			return
		}
		err = json.Unmarshal(b, out)
		if err == io.EOF {
			err = nil
		}
	default:
		if b, err = r.readBody(); err != nil {
			return
		}
		err = NewResponseError(r.StatusCode, string(b))
	}
	return
}

func encodeJson(data interface{}) (buf *bytes.Buffer, err error) {
	var encoder *json.Encoder
	buf = bytes.NewBufferString("")
	encoder = json.NewEncoder(buf)
	if data != nil {
		if err = encoder.Encode(data); err != nil {
			return
		}
	}
	return
}
