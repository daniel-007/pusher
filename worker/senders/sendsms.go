package senders

import (
	"bytes"
	"encoding/json"
	"fmt"
	pusherLib "github.com/Lupino/pusher"
	"github.com/Lupino/pusher/utils"
	"github.com/Lupino/pusher/worker"
	"log"
	"net/http"
	"net/url"
	"text/template"
	"time"
)

var apiRoot = "http://gw.api.taobao.com/router/rest"

// SMSSender a alidayu sms sender
type SMSSender struct {
	appKey    string
	appSecret string
	w         worker.Worker
}

// NewSMSSender new alidayu sms sender
func NewSMSSender(w worker.Worker, key, secret string) SMSSender {
	return SMSSender{
		appKey:    key,
		appSecret: secret,
		w:         w,
	}
}

type smsObject struct {
	PhoneNumber string `json:"phoneNumber"`
	Params      string `json:"params"`
	SignName    string `json:"signName"`
	Template    string `json:"template"`
	CreatedAt   int64  `json:"createdAt"`
}

// GetName for the periodic funcName
func (SMSSender) GetName() string {
	return "sendsms"
}

// Send message to pusher then return sendlater
func (s SMSSender) Send(pusher, data string, counter int) (int, error) {
	var (
		sms    smsObject
		err    error
		p      pusherLib.Pusher
		params string
		tpl    *template.Template
		buffer = bytes.NewBuffer(nil)
	)
	if err = json.Unmarshal([]byte(data), &sms); err != nil {
		log.Printf("json.Unmarshal() failed (%s)", err)
		return 0, nil
	}

	if p, err = s.w.GetAPI().GetPusher(pusher); err != nil {
		log.Printf("worker.API.GetPusher() failed (%s)", err)
		return 0, nil
	}

	if sms.PhoneNumber == "" {
		sms.PhoneNumber = p.PhoneNumber
	}

	if sms.PhoneNumber == "" {
		return 0, nil
	}

	params = sms.Params
	if tpl, err = template.New("smsParams").Parse(sms.Params); err != nil {
		log.Printf("template.New().Parse() failed (%s)", err)
	} else {
		if err = tpl.Execute(buffer, p); err != nil {
			log.Printf("template.Template.Execute() failed (%s)", err)
		} else {
			params = string(buffer.Bytes())
		}
	}

	if err = s.SendSMS(sms.PhoneNumber, params, sms.SignName, sms.Template); err != nil {
		log.Printf("senders.SMSSender.SendSMS() failed (%s)", err)
		log.Printf("senders.SMSSender.Send() retry send later (%ss)", 10*counter)
		return 10 * counter, nil
	}
	return 0, nil
}

// SendSMS message
func (s SMSSender) SendSMS(phoneNumber, smsParams, signName, template string) error {
	params := make(map[string]string)
	params["method"] = "alibaba.aliqin.fc.sms.num.send"
	params["app_key"] = s.appKey
	params["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	params["format"] = "json"
	params["v"] = "2.0"
	params["sign_method"] = "hmac"
	params["sms_type"] = "normal"
	params["sms_free_sign_name"] = signName
	params["rec_num"] = phoneNumber
	params["sms_param"] = smsParams
	params["sms_template_code"] = template
	params["sign"] = utils.HmacMD5(s.appSecret, params)

	var form = url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}

	rsp, err := http.PostForm(apiRoot, form)
	if err != nil {
		log.Printf("http.PostForm() failed (%s)", err)
		return err
	}
	defer rsp.Body.Close()

	decoder := json.NewDecoder(rsp.Body)
	var ret map[string]interface{}
	if err = decoder.Decode(&ret); err != nil {
		log.Printf("json.NewDecoder().Decode() failed (%s)", err)
		return err
	}
	errRsp, ok := ret["error_response"]
	if !ok {
		return nil
	}
	errRet, ok := errRsp.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unknow error (%s)", errRsp)
	}
	return fmt.Errorf("%s", errRet["sub_code"])
}
