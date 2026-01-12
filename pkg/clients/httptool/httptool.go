package httptool

import (
	"ai_web/test/config"
	"ai_web/test/pkg/tools"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ConnectionRefusedTag = "connection refused"

	HeaderContentType       = "Content-Type"
	HeaderContentTypeStream = "text/event-stream;charset=utf-8"
	HeaderContentCache      = "Cache-Control"
	HeaderContentCacheNo    = "no-cache"
	HeaderContentConnection = "Connection"
	HeaderContentKeepAlive  = "keep-alive"
	HeaderContentTransfer   = "Transfer-Encoding"
	HeaderContentChunked    = "chunked"
)

var replaceErrorMsg = map[string]string{
	ConnectionRefusedTag: "链接失败",
}

type ResponseMsg struct {
	Message string `json:"message"`
}

type HTTPClient struct {
	sync.RWMutex
	hc                http.Client
	baseAddr          string
	defaultRespReader HTTPResponseReader
	header            http.Header
	clientName        string
	//endpointMap       map[string]string
}

type HTTPResponseReader func(*http.Response, *http.Request, time.Time) ([]byte, error)

func NewHTTPClient(baseAddr, clientName string, timeout time.Duration, transport *http.Transport, defaultRespReader HTTPResponseReader) *HTTPClient {
	ret := &HTTPClient{
		baseAddr: "http://" + baseAddr,
		hc: http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		clientName: clientName,
	}
	if defaultRespReader == nil {
		ret.defaultRespReader = ret.readResponse
	} else {
		ret.defaultRespReader = defaultRespReader
	}
	return ret
}

func (hc *HTTPClient) SetHeader(key, value string) {
	hc.Lock()
	defer hc.Unlock()

	if hc.header == nil {
		hc.header = http.Header{}
	}

	hc.header.Set(key, value)
}

//func (hc *HTTPClient) SetEndPoint(url, value string) {
//	hc.Lock()
//	defer hc.Unlock()
//	if hc.endpointMap == nil {
//		hc.endpointMap = map[string]string{}
//	}
//	hc.endpointMap[url] = value
//}
//
//func (hc *HTTPClient) GetEndPoint(url string) string {
//	hc.RLock()
//	defer hc.RUnlock()
//	if hc.endpointMap == nil {
//		return url
//	}
//
//	v, ok := hc.endpointMap[url]
//	if !ok {
//		v = url
//	}
//
//	return v
//}

func (hc *HTTPClient) GetWithContext(ctx context.Context, url string) ([]byte, error) {
	return hc.fetchWithContext(ctx, "GET", url, nil, nil)
}
func (hc *HTTPClient) Get(url string) ([]byte, error) {
	return hc.fetch("GET", url, nil, nil)
}

func (hc *HTTPClient) GetWithHeaderAndContext(ctx context.Context, url string, header http.Header) ([]byte, http.Header, error) {
	return hc.fetchWithHeaderAndContext(ctx, "GET", url, nil, nil, header)
}

func (hc *HTTPClient) fetchWithHeaderAndContext(ctx context.Context, method, url string, body io.Reader, respReader HTTPResponseReader, header http.Header) ([]byte, http.Header, error) {
	targetURL := fmt.Sprintf("%v%v", hc.baseAddr, url)

	ok := config.GetInstance().GetBool(config.ClientsCommonRequestLog)
	now := time.Now()

	if ok && body != nil {
		b, _ := io.ReadAll(body)

		body = bytes.NewReader(b)
		log.Debugf("Sending %v request to %v", method, targetURL)
		log.Debugf("Body = %v", string(b))
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	req.Header = header
	resp, err := hc.hc.Do(req)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	defer tools.ErrorWithPrintContext(resp.Body.Close, "close response body")

	r := hc.getRespReader(respReader)
	responseBody, err := r(resp, req, now)
	return responseBody, resp.Header, err
}

func (hc *HTTPClient) GetWithBody(url string, obj interface{}) ([]byte, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return hc.fetch("GET", url, bytes.NewBuffer(body), nil)
}

func (hc *HTTPClient) GetWithBodyAndContext(ctx context.Context, url string, obj interface{}) ([]byte, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return hc.fetchWithContext(ctx, "GET", url, bytes.NewBuffer(body), nil)
}

func (hc *HTTPClient) GetParams(url string, params map[string][]string) ([]byte, error) {
	if len(params) == 0 {
		return hc.fetch("GET", url, nil, nil)
	}
	var paramSlice []string
	for key, valSlice := range params {
		for _, val := range valSlice {
			paramSlice = append(paramSlice, key+"="+val)
		}
	}
	url = url + "?" + strings.Join(paramSlice, "&")
	return hc.fetch("GET", url, nil, nil)

}

func (hc *HTTPClient) GetParamsWithContext(ctx context.Context, url string, params map[string][]string) ([]byte, error) {
	if len(params) == 0 {
		return hc.fetchWithContext(ctx, "GET", url, nil, nil)
	}
	var paramSlice []string
	for key, valSlice := range params {
		for _, val := range valSlice {
			paramSlice = append(paramSlice, key+"="+val)
		}
	}
	url = url + "?" + strings.Join(paramSlice, "&")
	return hc.fetchWithContext(ctx, "GET", url, nil, nil)

}

func (hc *HTTPClient) PostJSON(url string, obj interface{}) ([]byte, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hc.Post(url, bytes.NewBuffer(body))
}

func (hc *HTTPClient) PostJSONWithContext(ctx context.Context, url string, obj interface{}) ([]byte, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hc.PostWithContext(ctx, url, bytes.NewBuffer(body))
}

func (hc *HTTPClient) Post(url string, body io.Reader) ([]byte, error) {
	return hc.fetch(http.MethodPost, url, body, nil)
}

func (hc *HTTPClient) PostWithContext(ctx context.Context, url string, body io.Reader) ([]byte, error) {
	return hc.fetchWithContext(ctx, http.MethodPost, url, body, nil)
}

func (hc *HTTPClient) PutJSON(url string, obj interface{}) ([]byte, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hc.Put(url, bytes.NewBuffer(body))
}

func (hc *HTTPClient) PutJSONWithContext(ctx context.Context, url string, obj interface{}) ([]byte, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hc.PutWithContext(ctx, url, bytes.NewBuffer(body))
}

func (hc *HTTPClient) PostForm(url, key, value string) ([]byte, error) {
	return hc.form(http.MethodPost, url, key, value, nil)
}

func (hc *HTTPClient) PostFormWithContext(ctx context.Context, url, key, value string) ([]byte, error) {
	return hc.formWithContext(ctx, http.MethodPost, url, key, value, nil)
}

func (hc *HTTPClient) Put(url string, body io.Reader) ([]byte, error) {
	return hc.fetch(http.MethodPut, url, body, nil)
}

func (hc *HTTPClient) PutWithContext(ctx context.Context, url string, body io.Reader) ([]byte, error) {
	return hc.fetchWithContext(ctx, http.MethodPut, url, body, nil)
}

func (hc *HTTPClient) DeleteJSON(url string, obj interface{}) ([]byte, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hc.Delete(url, bytes.NewBuffer(body))
}

func (hc *HTTPClient) DeleteJSONWithContext(ctx context.Context, url string, obj interface{}) ([]byte, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hc.DeleteWithContext(ctx, url, bytes.NewBuffer(body))
}

func (hc *HTTPClient) Delete(url string, body io.Reader) ([]byte, error) {
	return hc.fetch(http.MethodDelete, url, body, nil)
}

func (hc *HTTPClient) DeleteWithContext(ctx context.Context, url string, body io.Reader) ([]byte, error) {
	return hc.fetchWithContext(ctx, http.MethodDelete, url, body, nil)
}

func (hc *HTTPClient) fetchWithContext(ctx context.Context, method, url string, body io.Reader, respReader HTTPResponseReader) ([]byte, error) {
	targetURL := fmt.Sprintf("%v%v", hc.baseAddr, url)

	ok := config.GetInstance().GetBool(config.ClientsCommonRequestLog)
	now := time.Now()

	if ok && body != nil {
		b, _ := io.ReadAll(body)

		body = bytes.NewReader(b)
		log.Debugf("Sending %v request to %v", method, targetURL)
		log.Debugf("Body = %v", string(b))
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	hc.RLock()
	req.Header = hc.header.Clone()
	hc.RUnlock()
	resp, err := hc.hc.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), ConnectionRefusedTag) {
			return nil, fmt.Errorf("%s模块: %s %s", hc.clientName, req.Host, replaceErrorMsg[ConnectionRefusedTag])
		}
		return nil, errors.WithStack(err)
	}
	// 写会request body方便传递
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	r := hc.getRespReader(respReader)
	return r(resp, req, now)
}

func (hc *HTTPClient) fetch(method, url string, body io.Reader, respReader HTTPResponseReader) ([]byte, error) {
	targetURL := fmt.Sprintf("%v%v", hc.baseAddr, url)

	ok := config.GetInstance().GetBool(config.ClientsCommonRequestLog)
	now := time.Now()

	if ok && body != nil {
		b, _ := io.ReadAll(body)

		body = bytes.NewReader(b)
		log.Debugf("Sending %v request to %v", method, targetURL)
		log.Debugf("Body = %v", string(b))
	}

	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	hc.RLock()
	req.Header = hc.header.Clone()
	hc.RUnlock()
	resp, err := hc.hc.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), ConnectionRefusedTag) {
			return nil, fmt.Errorf("%s模块: %s %s", hc.clientName, req.Host, replaceErrorMsg[ConnectionRefusedTag])
		}
		return nil, errors.WithStack(err)
	}
	// 写会request body方便传递
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	r := hc.getRespReader(respReader)
	return r(resp, req, now)
}

func (hc *HTTPClient) form(method, formURL, key, value string, respReader HTTPResponseReader) ([]byte, error) {
	targetURL := fmt.Sprintf("%v%v", hc.baseAddr, formURL)
	// log.Debugf("Sending %v request to %v", method, targetURL)
	// log.Debugf("form key =%v value=%v", key, value)

	data := url.Values{}
	data.Set(key, value)

	now := time.Now()
	req, err := http.NewRequest(method, targetURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := hc.hc.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), ConnectionRefusedTag) {
			return nil, fmt.Errorf("%s模块: %s %s", hc.clientName, req.Host, replaceErrorMsg[ConnectionRefusedTag])
		}
		return nil, errors.WithStack(err)
	}

	r := hc.getRespReader(respReader)

	return r(resp, req, now)
}

func (hc *HTTPClient) formWithContext(ctx context.Context, method, formURL, key, value string, respReader HTTPResponseReader) ([]byte, error) {
	targetURL := fmt.Sprintf("%v%v", hc.baseAddr, formURL)
	// log.Debugf("Sending %v request to %v", method, targetURL)
	// log.Debugf("form key =%v value=%v", key, value)

	data := url.Values{}
	data.Set(key, value)

	now := time.Now()
	req, err := http.NewRequestWithContext(ctx, method, targetURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := hc.hc.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), ConnectionRefusedTag) {
			return nil, fmt.Errorf("%s模块: %s %s", hc.clientName, req.Host, replaceErrorMsg[ConnectionRefusedTag])
		}
		return nil, errors.WithStack(err)
	}

	r := hc.getRespReader(respReader)

	return r(resp, req, now)
}

func (hc *HTTPClient) fetchWithContentType(method, url, contentType string, body io.Reader, respReader HTTPResponseReader) ([]byte, error) {
	targetURL := fmt.Sprintf("%v%v", hc.baseAddr, url)

	ok := config.GetInstance().GetBool(config.ClientsCommonRequestLog)

	if ok {
		log.Debugf("Sending %v requst to %v", method, targetURL)
		log.Debugf("Body = %v", body)
	}

	now := time.Now()
	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := hc.hc.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), ConnectionRefusedTag) {
			return nil, fmt.Errorf("%s模块: %s %s", hc.clientName, req.Host, replaceErrorMsg[ConnectionRefusedTag])
		}
		return nil, errors.WithStack(err)
	}
	r := hc.getRespReader(respReader)
	return r(resp, req, now)
}

func (hc *HTTPClient) fetchWithContentTypeAndContext(ctx context.Context, method, url, contentType string, body io.Reader, respReader HTTPResponseReader) ([]byte, error) {
	targetURL := fmt.Sprintf("%v%v", hc.baseAddr, url)

	ok := config.GetInstance().GetBool(config.ClientsCommonRequestLog)

	if ok {
		log.Debugf("Sending %v requst to %v", method, targetURL)
		log.Debugf("Body = %v", body)
	}

	now := time.Now()
	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := hc.hc.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), ConnectionRefusedTag) {
			return nil, fmt.Errorf("%s模块: %s %s", hc.clientName, req.Host, replaceErrorMsg[ConnectionRefusedTag])
		}
		return nil, errors.WithStack(err)
	}
	r := hc.getRespReader(respReader)
	return r(resp, req, now)
}

func (hc *HTTPClient) PostFile(ctx context.Context, url, filename, filePath string, limit int) ([]byte, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer tools.ErrorWithPrintContext(file.Close, "close file %s", filePath)

	//创建一个模拟的form中的一个选项,这个form项现在是空的
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	//关键的一步操作, 设置文件的上传参数叫uploadfile, 文件名是filename,
	//相当于现在还没选择文件, form项里选择文件的选项
	fileWriter, err := bodyWriter.CreateFormFile("image", filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	//iocopy 这里相当于选择了文件,将文件放到form中
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if fileWriter, err = bodyWriter.CreateFormField("limit"); err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err = fmt.Fprintf(fileWriter, "%v", limit); err != nil {
		return nil, errors.WithStack(err)
	}

	//获取上传文件的类型,multipart/form-data; boundary=...
	contentType := bodyWriter.FormDataContentType()

	//这个很关键,必须这样写关闭,不能使用defer关闭,不然会导致错误
	if err = bodyWriter.Close(); err != nil {
		return nil, errors.WithStack(err)
	}

	//发送post请求到服务端
	resp, err := hc.fetchWithContentType("POST", url, contentType, bodyBuf, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return resp, nil
}

func (hc *HTTPClient) PostFileWithContext(ctx context.Context, url, filename, filePath string, limit int) ([]byte, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer tools.ErrorWithPrintContext(file.Close, "close file %s", filePath)

	//创建一个模拟的form中的一个选项,这个form项现在是空的
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	//关键的一步操作, 设置文件的上传参数叫uploadfile, 文件名是filename,
	//相当于现在还没选择文件, form项里选择文件的选项
	fileWriter, err := bodyWriter.CreateFormFile("image", filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	//iocopy 这里相当于选择了文件,将文件放到form中
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if fileWriter, err = bodyWriter.CreateFormField("limit"); err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err = fmt.Fprintf(fileWriter, "%v", limit); err != nil {
		return nil, errors.WithStack(err)
	}

	//获取上传文件的类型,multipart/form-data; boundary=...
	contentType := bodyWriter.FormDataContentType()

	//这个很关键,必须这样写关闭,不能使用defer关闭,不然会导致错误
	if err = bodyWriter.Close(); err != nil {
		return nil, errors.WithStack(err)
	}

	//发送post请求到服务端
	resp, err := hc.fetchWithContentTypeAndContext(ctx, "POST", url, contentType, bodyBuf, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return resp, nil
}

func (hc *HTTPClient) getRespReader(respReader HTTPResponseReader) HTTPResponseReader {
	if respReader != nil {
		return respReader
	}
	return hc.defaultRespReader
}

func (hc *HTTPClient) readResponse(resp *http.Response, req *http.Request, startTime time.Time) ([]byte, error) {
	defer tools.ErrorWithPrintContext(resp.Body.Close, "close response body")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var respStr string
	if len(bodyBytes) > 1024*100 {
		respStr = fmt.Sprintf("resp大小: %v", len(bodyBytes))
	} else {
		respStr = string(bodyBytes)
	}

	ok := config.GetInstance().GetBool(config.ClientsCommonRequestLog)
	if ok {
		log.Debugf("Got response from %v %v, status code = %d, body = %v took = %v", req.Method, req.URL, resp.StatusCode, respStr, time.Since(startTime))
	}

	if time.Since(startTime) > 800*time.Millisecond {
		log.Infof("TimeConsuming: from %v %v, request body = %v status code = %d, response body = %v took = %v\n", req.Method, req.URL, req.Body, resp.StatusCode, respStr, time.Since(startTime))
	}

	if resp.StatusCode/100 != 2 {
		errMsg := fmt.Errorf("HTTP request to %v %v failed with status code %d response:%v", req.Method, req.URL, resp.StatusCode, string(bodyBytes))
		if string(bodyBytes) == "" {
			return bodyBytes, errMsg
		}
		var result = new(ResponseMsg)
		if err = json.Unmarshal(bodyBytes, result); err != nil {
			return bodyBytes, errMsg
		}
		return bodyBytes, fmt.Errorf("%s", result.Message)
	}
	return bodyBytes, nil
}
