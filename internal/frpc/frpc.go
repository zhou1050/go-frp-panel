package frpc

import (
	"context"
	"fmt"
	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/system"
	"github.com/xxl6097/glog/glog"
	_ "github.com/xxl6097/go-frp-panel/assets/frpc"
	"github.com/xxl6097/go-frp-panel/internal/comm"
	"github.com/xxl6097/go-frp-panel/internal/comm/iface"
	"github.com/xxl6097/go-frp-panel/pkg/utils"
	"github.com/xxl6097/go-service/gservice/gore"
	utils2 "github.com/xxl6097/go-service/gservice/utils"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type frpClient struct {
	svr            *client.Service
	configFilePath string
	cfg            *v1.ClientCommonConfig
	proxyCfg       []v1.ProxyConfigurer
	visitorCfg     []v1.VisitorConfigurer
}
type frpc struct {
	install   gore.IGService
	upgrade   iface.IComm
	cls       *frpClient
	cfgBuffer *comm.BufferConfig
	svrs      map[string]*frpClient
}

func NewFrpc(i gore.IGService) (*frpc, error) {
	baseDir, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cfgFilePath := filepath.Join(filepath.Dir(baseDir), "config.toml")
	if !utils2.FileExists(cfgFilePath) {
		return nil, fmt.Errorf("config file %s not exists", cfgFilePath)
	}
	cfg, proxyCfgs, visitorCfgs, isLegacyFormat, err := config.LoadClientConfig(cfgFilePath, true)
	if err != nil {
		return nil, fmt.Errorf("load config file %s not exists", cfgFilePath)
	}
	if isLegacyFormat {
		fmt.Printf("WARNING: ini format is deprecated and the support will be removed in the future, " +
			"please use yaml/json/toml format instead!\n")
	}

	warning, err := validation.ValidateAllClientConfig(cfg, proxyCfgs, visitorCfgs)
	if warning != nil {
		fmt.Printf("WARNING: %v\n", warning)
	}
	if err != nil {
		return nil, fmt.Errorf("ValidateAllClientConfig config file %v err", err)
	}

	system.EnableCompatibilityMode()
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)
	svr, err := client.NewService(client.ServiceOptions{
		Common:         cfg,
		ProxyCfgs:      proxyCfgs,
		VisitorCfgs:    visitorCfgs,
		ConfigFilePath: cfgFilePath,
	})
	if err != nil {
		return nil, err
	}
	cfgModel := GetCfgModel()
	this := &frpc{
		install: i,
		svrs:    make(map[string]*frpClient),
		upgrade: comm.NewCommApi(i, cfgModel),
		cls: &frpClient{
			svr:            svr,
			configFilePath: cfgFilePath,
			cfg:            cfg,
			proxyCfg:       proxyCfgs,
			visitorCfg:     visitorCfgs,
		},
	}
	if cfgModel.Cfg != nil {
		this.cfgBuffer = cfgModel.Cfg
	}

	shouldGracefulClose := cfg.Transport.Protocol == "kcp" || cfg.Transport.Protocol == "quic"
	if shouldGracefulClose {
		go this.handleTermSignal(svr)
	}

	webServer := utils.GetPointerInstance[httppkg.Server]("webServer", svr)
	if webServer == nil {
		return nil, fmt.Errorf("can't find webServer")
	}
	webServer.RouteRegister(this.adminHandlers)
	go this.runMultipleClients(filepath.Join(filepath.Dir(baseDir), "config"))
	return this, nil
}

func (this *frpc) handleTermSignal(svr *client.Service) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	svr.GracefulClose(500 * time.Millisecond)
}

func (this *frpc) Run() error {
	err := this.cls.svr.Run(context.Background())
	if err != nil {
		glog.Errorf("frpc run error: %v", err)
	}
	return err
}
