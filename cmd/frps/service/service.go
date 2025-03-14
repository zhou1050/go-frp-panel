package service

import (
	"encoding/json"
	"fmt"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/kardianos/service"
	"github.com/xxl6097/glog/glog"
	"github.com/xxl6097/go-frp-panel/internal/comm/iface"
	"github.com/xxl6097/go-frp-panel/internal/frps"
	"github.com/xxl6097/go-frp-panel/pkg"
	"github.com/xxl6097/go-service/gservice/gore"
	"github.com/xxl6097/go-service/gservice/ukey"
	utils2 "github.com/xxl6097/go-service/gservice/utils"
	"path/filepath"
)

type Service struct {
	ifrps iface.IFrps
}

func (s Service) OnInit() *service.Config {
	return &service.Config{
		Name:        pkg.AppName,
		DisplayName: pkg.DisplayName,
		Description: pkg.Description,
	}
}

func (s Service) OnStop(ss service.Service) {
	s.ifrps.Close()
}

func (s Service) ShutDown(ss service.Service) {
	s.ifrps.Close()
}

func (s Service) OnVersion() string {
	fmt.Println(string(ukey.GetBuffer()))
	//这里需要打印config中buffer原始信息
	ver := fmt.Sprintf("frps version:%s", version.Full())
	pkg.Version()
	return ver
}

func (this Service) OnRun(i gore.IGService) error {
	frps.Assert()
	glog.Printf("启动frps_%s\n", pkg.AppVersion)
	cfg := frps.GetCfgModel()
	if cfg == nil {
		return fmt.Errorf("程序配置文件未初始化")
	}
	conf := frps.GetCfgModel().Frps
	content, err := json.Marshal(conf)
	if err != nil {
		glog.Error(err)
		return err
	}
	svv, err := frps.NewFrps(content, i)
	if err != nil {
		glog.Error("启动frps失败", err)
		glog.Printf("启动frps_%s失败\n", pkg.AppVersion)
		glog.Println(conf)
		return err
	}
	this.ifrps = svv
	svv.Run()
	return err
}

func (this Service) GetAny(binDir string) any {
	return this.menu()
}

//func (s Service) OnUpgrade(oldBinPath string, newFileUrlOrLocalPath string) (bool, []string) {
//	//1、读取老文件特征数据；
//	//2、下载新文件
//	//3、替换新文件特征数据
//	//4、数据写到安装目录地址（oldBinPath）
//	cfgBufferBytes := ukey.GetCfgBufferFromFile(oldBinPath)
//	if cfgBufferBytes == nil {
//		return false, nil
//	}
//	glog.Debug("获取配置数据成功", len(cfgBufferBytes))
//	if _, err := os.Stat(oldBinPath); !os.IsNotExist(err) {
//		err := os.Remove(oldBinPath)
//		if err != nil {
//			glog.Error("删除失败", oldBinPath)
//			return false, nil
//		}
//	}
//	var newFilePath string
//	if utils2.FileExists(newFileUrlOrLocalPath) {
//		newFilePath = newFileUrlOrLocalPath
//	} else if utils2.IsURL(newFileUrlOrLocalPath) {
//		glog.Debug("下载文件", newFileUrlOrLocalPath)
//		temp, err := utils.DownLoad(newFileUrlOrLocalPath)
//		if err != nil {
//			glog.Error("下载失败", err)
//			return false, nil
//		}
//		glog.Debug("下载成功.", temp)
//		newFilePath = temp
//	}
//	if newFilePath != "" {
//		oldBuffer := ukey.GetBuffer()
//		err := utils.GenerateBin(newFilePath, oldBinPath, oldBuffer, cfgBufferBytes)
//		if err != nil {
//			glog.Error("签名错误：", err)
//			return false, nil
//		}
//		return true, nil
//	}
//	return false, nil
//}
//
//func (s Service) OnInstall(binPath string) (bool, []string) {
//	if frps.IsInit() == nil {
//		return false, nil
//	}
//	cfg := s.menu()
//	//cfg.Frps.Complete()
//	newBufferBytes, err := ukey.GenConfig(cfg, false)
//	if err != nil {
//		panic(fmt.Errorf("构建签名信息错误: %v", err))
//	}
//	//glog.Printf("--->%s\n", string(newBufferBytes))
//	currentBinPath, err := os.Executable()
//	if err != nil {
//		glog.Fatal("os.Executable() error", err)
//	}
//	if utils2.FileExists(binPath) {
//		utils.Delete(binPath, "旧运行文件")
//	}
//	//安装程序，需要对程序进行签名，那么需要传入两个参数：
//	//1、最原始的key；
//	//2、需写入的data
//	buffer := ukey.GetBuffer()
//	glog.Info("buffer大小", len(buffer))
//	err = utils.GenerateBin(currentBinPath, binPath, buffer, newBufferBytes)
//	if err != nil {
//		glog.Fatal("签名错误：", err)
//	}
//	return false, nil
//}

func (this *Service) menu() *frps.CfgModel {
	bindPort := utils2.InputInt("请输入Frps服务器绑定端口：")
	adminPort := utils2.InputInt("请输入管理后台端口：")
	addr := utils2.InputStringEmpty("请输入管理后台地址(默认0.0.0.0)：", "0.0.0.0")
	username := utils2.InputStringEmpty("请输入管理后台用户名(admin)：", "admin")
	password := utils2.InputString("请输入管理后台密码：")
	temp := glog.GetCrossPlatformDataDir(pkg.AppName, "frps", "log")
	cfg := &frps.CfgModel{
		Frps: v1.ServerConfig{
			BindPort: bindPort,
			HTTPPlugins: []v1.HTTPPluginOptions{
				{
					Name: "frps-panel",
					Addr: fmt.Sprintf("%s:%d", addr, adminPort),
					Path: "/handler",
					Ops:  []string{"Login", "NewWorkConn", "NewUserConn", "NewProxy", "Ping"},
				},
			},
			WebServer: v1.WebServerConfig{
				User:     username,
				Password: password,
				Port:     adminPort,
				Addr:     addr,
			},
			Log: v1.LogConfig{
				To:      filepath.Join(temp, "frps.log"),
				MaxDays: 15,
			},
		},
	}
	return cfg
}
