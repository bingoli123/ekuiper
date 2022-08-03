package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type kongError struct {
	Code    *int64  `json:"code,omitempty"`
	Message *string `json:"message,omitempty"`
	Name    *string `json:"name,omitempty"`
}

type addKongService struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
	URL  string   `json:"url,omitempty"`
}

type addKongRoute struct {
	Name      string   `json:"name"`
	Protocols []string `json:"protocols"`
	Tags      []string `json:"tags"`
	Paths     []string `json:"paths"`
}

type addKongPlugin struct {
	Protocols []string `json:"protocols"`
	Enabled   bool     `json:"enabled"`
	Tags      []string `json:"tags"`
	Name      string   `json:"name"`
	//Config    interface{} `json:"config,omitempty"`
}

/*
type basicAuthCfg struct {
	Anonymous       *string `json:"anonymous"`
	HideCredentials bool    `json:"hide_credentials"`
}
*/

func KongInit(kongLocalManageHost string, kongLocalManagePort int, httpListenPort int, KongPlugins []string) error {
	kongAdminPort := strconv.Itoa(kongLocalManagePort)
	headers := map[string]string{"Content-Type": "application/json"}
	var strBuilder strings.Builder
	strBuilder.WriteString("http://connection-coordinator.service.consul")
	strBuilder.WriteString(":")
	strBuilder.WriteString(strconv.Itoa(httpListenPort))
	strBuilder.WriteString("/connection-coordinator")
	var addservice = new(addKongService)
	addservice.Name = "connection-coordinator"
	addservice.URL = strBuilder.String()
	addservice.Tags = append(addservice.Tags, "connection-coordinator", "3.0")

	strBuilder.Reset()
	strBuilder.WriteString("http://")
	strBuilder.WriteString(kongLocalManageHost)
	strBuilder.WriteString(":")
	strBuilder.WriteString(kongAdminPort)
	strBuilder.WriteString("/services/connection-coordinator")
	resBody, resStatus, resErr := sendMsgToHttpUrl(strBuilder.String(), http.MethodPut, headers, addservice) // 注册服务
	if resErr != nil {
		return resErr
	}

	if resStatus != http.StatusOK {
		var errorResponse = new(kongError)
		errorResponse.Code = new(int64)
		errorResponse.Name = new(string)
		errorResponse.Message = new(string)
		if err := json.Unmarshal(resBody, errorResponse); err != nil {
			return err
		}
		return errors.New(fmt.Sprintf("kong add service faild, code=%d, name=%s, msg=%s", *errorResponse.Code, *errorResponse.Name, *errorResponse.Message))
	}

	var addRoute = new(addKongRoute)
	addRoute.Name = "connection-coordinator-inner-route"
	addRoute.Protocols = append(addRoute.Protocols, "http")
	addRoute.Paths = append(addRoute.Paths, "/connection-coordinator")
	addRoute.Tags = append(addRoute.Tags, "connection-coordinator", "3.0", "inner")

	strBuilder.Reset()
	strBuilder.WriteString("http://")
	strBuilder.WriteString(kongLocalManageHost)
	strBuilder.WriteString(":")
	strBuilder.WriteString(kongAdminPort)
	strBuilder.WriteString("/services/connection-coordinator/routes/connection-coordinator-inner-route")
	resBody, resStatus, resErr = sendMsgToHttpUrl(strBuilder.String(), http.MethodPut, headers, addRoute) // 注册内部路由
	if resErr != nil {
		return resErr
	}

	if resStatus != http.StatusOK {
		var errorResponse = new(kongError)
		errorResponse.Code = new(int64)
		errorResponse.Name = new(string)
		errorResponse.Message = new(string)
		if err := json.Unmarshal(resBody, errorResponse); err != nil {
			return err
		}

		return errors.New(fmt.Sprintf("kong add service route faild, code=%d, name=%s, msg=%s", *errorResponse.Code, *errorResponse.Name, *errorResponse.Message))
	}

	addRoute = new(addKongRoute)
	addRoute.Name = "connection-coordinator-outer-route"
	addRoute.Protocols = append(addRoute.Protocols, "http")
	addRoute.Paths = append(addRoute.Paths, "/ivideo/connection-coordinator")
	addRoute.Tags = append(addRoute.Tags, "connection-coordinator", "3.0", "outer")

	strBuilder.Reset()
	strBuilder.WriteString("http://")
	strBuilder.WriteString(kongLocalManageHost)
	strBuilder.WriteString(":")
	strBuilder.WriteString(kongAdminPort)
	strBuilder.WriteString("/services/connection-coordinator/routes/connection-coordinator-outer-route")
	resBody, resStatus, resErr = sendMsgToHttpUrl(strBuilder.String(), http.MethodPut, headers, addRoute) // 注册外部路由
	if resErr != nil {
		return resErr
	}

	if resStatus != http.StatusOK {
		var errorResponse = new(kongError)
		errorResponse.Code = new(int64)
		errorResponse.Name = new(string)
		errorResponse.Message = new(string)
		if err := json.Unmarshal(resBody, errorResponse); err != nil {
			return err
		}

		return errors.New(fmt.Sprintf("kong add service route faild, code=%d, name=%s, msg=%s", *errorResponse.Code, *errorResponse.Name, *errorResponse.Message))
	}

	// 适配支持的插件
	//pluginsMap := map[string]interface{}{
	//	"basic-auth": basicAuthCfg{
	//		Anonymous:       nil,
	//		HideCredentials: false,
	//	},
	//}
	//
	for _, plugin := range KongPlugins { //加载路由插件
		//cfg, ok := pluginsMap[plugin]
		//if !ok {
		//	strBuilder.Reset()
		//	strBuilder.WriteString("unsupport plugin ")
		//	strBuilder.WriteString(plugin)
		//	return errors.New(strBuilder.String())
		//}
		routePlugin := new(addKongPlugin)
		//routePlugin.Config = cfg
		routePlugin.Enabled = true
		routePlugin.Name = plugin
		routePlugin.Tags = append(routePlugin.Tags, "connection-coordinator", "3.0")
		routePlugin.Protocols = append(routePlugin.Protocols, "http", "https")

		strBuilder.Reset()
		strBuilder.WriteString("http://")
		strBuilder.WriteString(strconv.Itoa(kongLocalManagePort))
		strBuilder.WriteString(":")
		strBuilder.WriteString(kongAdminPort)
		//strBuilder.WriteString("/routes/connection-coordinator-outer-route/plugins")
		strBuilder.WriteString("/services/connection-coordinator/plugins")
		resBody, resStatus, resErr = sendMsgToHttpUrl(strBuilder.String(), http.MethodPost, headers, routePlugin) // 添加插件
		if resErr != nil {
			return resErr
		}

		if resStatus != http.StatusCreated && resStatus != http.StatusConflict {
			var errorResponse = new(kongError)
			errorResponse.Code = new(int64)
			errorResponse.Name = new(string)
			errorResponse.Message = new(string)
			if err := json.Unmarshal(resBody, errorResponse); err != nil {
				return err
			}

			return errors.New(fmt.Sprintf("kong add service route plugin faild, code=%d, name=%s, msg=%s", *errorResponse.Code, *errorResponse.Name, *errorResponse.Message))
		}
	}

	return nil
}

func KongUninit(kongLocalManageHost string, kongLocalManagePort int) {
	kongAdminPort := strconv.Itoa(kongLocalManagePort)
	var strBuilder strings.Builder
	strBuilder.WriteString("http://")
	strBuilder.WriteString(kongLocalManageHost)
	strBuilder.WriteString(":")
	strBuilder.WriteString(kongAdminPort)
	strBuilder.WriteString("/services/connection-coordinator/routes/connection-coordinator-outer-route")
	resBody, resStatus, resErr := sendMsgToHttpUrl(strBuilder.String(), http.MethodDelete, nil, nil) // 删除外部路由,上面插件跟着删除
	if resErr != nil {
		fmt.Printf("call delete kong service route failed: %s \n", resErr.Error())
	} else {
		if resStatus != http.StatusNoContent && resStatus != http.StatusNotFound {
			var errorResponse = new(kongError)
			errorResponse.Code = new(int64)
			errorResponse.Name = new(string)
			errorResponse.Message = new(string)
			if err := json.Unmarshal(resBody, errorResponse); err != nil {
				fmt.Printf("json unmarshal failed, body = %s", string(resBody))
			} else {
				fmt.Printf("kong delete service route faild, msg=%s", *errorResponse.Message)
			}
		}
	}

	strBuilder.Reset()
	strBuilder.WriteString("http://")
	strBuilder.WriteString(kongLocalManageHost)
	strBuilder.WriteString(":")
	strBuilder.WriteString(kongAdminPort)
	strBuilder.WriteString("/services/connection-coordinator/routes/connection-coordinator-inner-route")
	resBody, resStatus, resErr = sendMsgToHttpUrl(strBuilder.String(), http.MethodDelete, nil, nil) // 删除内部路由
	if resErr != nil {
		fmt.Printf("call delete kong service route failed: %s", resErr.Error())
	} else {
		if resStatus != http.StatusNoContent && resStatus != http.StatusNotFound {
			var errorResponse = new(kongError)
			errorResponse.Code = new(int64)
			errorResponse.Name = new(string)
			errorResponse.Message = new(string)
			if err := json.Unmarshal(resBody, errorResponse); err != nil {
				fmt.Printf("json unmarshal failed, body = %s", string(resBody))
			} else {
				fmt.Printf("kong delete service route faild, msg=%s", *errorResponse.Message)
			}
		}
	}

	strBuilder.Reset()
	strBuilder.WriteString("http://")
	strBuilder.WriteString(kongLocalManageHost)
	strBuilder.WriteString(":")
	strBuilder.WriteString(kongAdminPort)
	strBuilder.WriteString("/services/connection-coordinator")
	resBody, resStatus, resErr = sendMsgToHttpUrl(strBuilder.String(), http.MethodDelete, nil, nil) // 删除服务
	if resErr != nil {
		fmt.Printf("call delete kong service failed: %s", resErr.Error())
	} else {
		if resStatus != http.StatusNoContent {
			var errorResponse = new(kongError)
			errorResponse.Code = new(int64)
			errorResponse.Name = new(string)
			errorResponse.Message = new(string)
			if err := json.Unmarshal(resBody, errorResponse); err != nil {
				fmt.Printf("json unmarshal failed, body = %s", string(resBody))
			} else {
				fmt.Printf("kong delete service faild, msg=%s", *errorResponse.Message)
			}
		}
	}
}

func sendMsgToHttpUrl(url string, method string, headers map[string]string, req interface{}) ([]byte, int, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("new request failed, err: \n" + identifyPanic())
		}
	}()

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Second*3)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second * 6,
		},
	}
	defer client.CloseIdleConnections()

	httpClientReq, err := http.NewRequest(method, url, strings.NewReader(string(reqBytes)))
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	for k, v := range headers {
		//httpClientReq.Header.Set(k, v) // HTTP响应头字段会自动将首字母和“-”后的第一个字母转换为大写，其余转换为小写
		httpClientReq.Header[k] = []string{v}
	}

	resp, err := client.Do(httpClientReq)
	defer func() {
		if nil != resp && nil != resp.Body {
			_ = resp.Body.Close()
		}
	}()
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	return body, resp.StatusCode, nil
}

func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}
