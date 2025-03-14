package frps

import (
	"encoding/json"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/xxl6097/glog/glog"
	"github.com/xxl6097/go-frp-panel/pkg"
	"github.com/xxl6097/go-frp-panel/pkg/utils"
	"github.com/xxl6097/go-service/gservice/ukey"
	"os"
)

var cfgData *CfgModel
var cfgBytes []byte

type CfgModel struct {
	Frps v1.ServerConfig `json:"frps"`
	Data any             `json:"data"`
}

func Test(c *CfgModel) {
	cfgData = c
}

func load() error {
	defer glog.Flush()
	byteArray, err := ukey.Load()
	if err != nil {
		//glog.Error(err)
		return err
	}
	cfgBytes = byteArray
	c := CfgModel{}
	err = json.Unmarshal(cfgBytes, &c)
	if err != nil {
		glog.Println("cfgBytes解析错误", err)
		return err
	}
	cfgData = &c
	glog.Printf("%d 配置加载成功：%+v\n", os.Getpid(), cfgData)
	pkg.Version()
	return nil
}

func GetCfgModel() *CfgModel {
	return cfgData
}

func PrintCfg() {
	if cfgBytes != nil {
		glog.Println(string(cfgBytes))
	}
}

func IsInit() error {
	//glog.Println("IsInit")
	defer glog.Flush()
	err := load()
	if err != nil {
		//glog.Println(err)
		return err
	}
	return nil
}

func Assert() {
	//glog.Println("Assert")
	if IsInit() != nil {
		if utils.IsMacOs() {
			return
		}
		os.Exit(0)
	}
}
