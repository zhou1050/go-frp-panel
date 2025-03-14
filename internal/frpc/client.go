package frpc

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/xxl6097/glog/glog"
	"github.com/xxl6097/go-frp-panel/pkg/utils"
	utils2 "github.com/xxl6097/go-service/gservice/utils"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func (this *frpc) runMultipleClients(cfgDir string) {
	err := filepath.WalkDir(cfgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".toml" {
			return nil
		}
		time.Sleep(time.Millisecond)
		err = this.runClient(path)
		if err != nil {
			glog.Errorf("创建客户端【%s】失败:%v", d.Name(), err)
		} else {
			glog.Infof("创建客户端【%s】成功", d.Name())
		}
		return err
	})
	if err != nil {
		glog.Error(err)
	}
}

func (this *frpc) startService(
	cfg *v1.ClientCommonConfig,
	proxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	cfgFile string,
) error {
	cfg.WebServer = v1.WebServerConfig{}
	if cfg.Log.To == "" {
		temp := filepath.Join(glog.GetAppLogDir(), cfg.User, "app.log")
		cfg.Log = v1.LogConfig{
			To:      temp,
			MaxDays: 7,
		}
	}

	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)

	if cfgFile != "" {
		log.Infof("start frpc service for config file [%s]", cfgFile)
		defer log.Infof("frpc service for config file [%s] stopped", cfgFile)
	}

	svr, err := client.NewService(client.ServiceOptions{
		Common:         cfg,
		ProxyCfgs:      proxyCfgs,
		VisitorCfgs:    visitorCfgs,
		ConfigFilePath: cfgFile,
	})
	if err != nil {
		return err
	}

	name := path.Base(cfgFile)
	this.svrs[name] = &frpClient{
		svr:            svr,
		cfg:            cfg,
		proxyCfg:       proxyCfgs,
		visitorCfg:     visitorCfgs,
		configFilePath: cfgFile,
	}
	glog.Debug("create frpc", name)
	shouldGracefulClose := cfg.Transport.Protocol == "kcp" || cfg.Transport.Protocol == "quic"
	// Capture the exit signal if we use kcp or quic.
	if shouldGracefulClose {
		go this.handleTermSignal(svr)
	}
	e := svr.Run(context.Background())
	if e != nil {
		glog.Error(e)
	}
	//因为Run是阻塞的，能执行到这一行，说明失败了
	delete(this.svrs, name)
	return e
}

func (this *frpc) deleteClient(cfgFilePath string) error {
	name := path.Base(cfgFilePath)
	glog.Debug("delete frpc", name)
	cls := this.svrs[name]
	if cls == nil {
		return fmt.Errorf("can't find client")
	}
	svr := cls.svr
	if svr == nil {
		return fmt.Errorf("can't find service")
	}
	svr.Close()
	svr.GracefulClose(100 * time.Millisecond)
	//svr.StatusExporter().GetProxyStatus()
	utils.Delete(cfgFilePath, fmt.Sprintf("客户端:%s", cfgFilePath))
	return nil
}

func (this *frpc) statusClient(cfgFilePath string) ([]byte, error) {
	name := path.Base(cfgFilePath)
	glog.Debug("status frpc", name)
	cls := this.svrs[name]
	if cls == nil {
		return nil, fmt.Errorf("客户端未创建")
	}
	svr := cls.svr
	if svr == nil {
		return nil, fmt.Errorf("客户端服务未创建")
	}
	ctl := utils.GetPointerInstance[client.Control]("ctl", svr)
	if ctl == nil {
		return nil, fmt.Errorf("没有找到服务控制器")
	}
	pm := utils.GetPointerInstance[proxy.Manager]("pm", ctl)
	if pm == nil {
		return nil, fmt.Errorf("没有找到服务代理器")
	}
	var (
		buf []byte
		res client.StatusResp = make(map[string][]client.ProxyStatusResp)
	)
	ps := pm.GetAllProxyStatus()
	glog.Debug("GetAllProxyStatus", len(ps))
	for _, status := range ps {
		res[status.Type] = append(res[status.Type], client.NewProxyStatusResp(status, cls.cfg.ServerAddr))
	}

	for _, arrs := range res {
		if len(arrs) <= 1 {
			continue
		}
		slices.SortFunc(arrs, func(a, b client.ProxyStatusResp) int {
			return cmp.Compare(a.Name, b.Name)
		})
	}
	glog.Infof("Http response [/api/status]")
	buf, _ = json.Marshal(&res)
	return buf, nil
}

func (this *frpc) updateClient(cfgFilePath string) error {
	name := path.Base(cfgFilePath)
	glog.Debug("update frpc", name)
	cls := this.svrs[name]
	if cls == nil {
		return fmt.Errorf("can't find client")
	}
	svr := cls.svr
	if svr == nil {
		return fmt.Errorf("can't find service")
	}
	cliCfg, proxyCfgs, visitorCfgs, _, err := config.LoadClientConfig(cfgFilePath, true)
	if err != nil {
		return fmt.Errorf("reload frpc config error: %v", err)
	}
	if _, err := validation.ValidateAllClientConfig(cliCfg, proxyCfgs, visitorCfgs); err != nil {
		return fmt.Errorf("validate frpc proxy config error: %v", err)
	}

	if err := svr.UpdateAllConfigurer(proxyCfgs, visitorCfgs); err != nil {
		return fmt.Errorf("update frpc proxy config error: %v", err)
	}
	cls.cfg = cliCfg
	cls.proxyCfg = proxyCfgs
	cls.visitorCfg = visitorCfgs
	return nil
}
func (this *frpc) getPort(i interface{}) int {
	switch v := i.(type) {
	case *v1.TCPProxyConfig:
		fmt.Printf("Received an TCPProxyConfig.RemotePort: %d\n", v.RemotePort)
		return v.RemotePort
	default:
		fmt.Println()
	}
	return 0
}

func (this *frpc) getTcpProxyArray(name string) []int {
	glog.Debug("info frpc", name)
	if name == "" {
		//主客户端
		ports := this.cfgBuffer.ParsePorts()
		for _, c := range this.cls.proxyCfg {
			port := this.getPort(c)
			if port > 0 {
				ports = utils.RemoveSlice[int](ports, port)
			}
		}
		return ports
	}
	return nil
}

func (this *frpc) runClient(cfgFilePath string) error {
	cfg, proxyCfgs, visitorCfgs, isLegacyFormat, err := config.LoadClientConfig(cfgFilePath, true)
	if err != nil {
		return err
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
		return err
	}
	e, _ := utils2.BlockingFunction[error](context.Background(), time.Second*3, func() error {
		return this.startService(cfg, proxyCfgs, visitorCfgs, cfgFilePath)
	})
	if e == nil {
	}
	return e
}
