package frps

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/xxl6097/glog/glog"
	"github.com/xxl6097/go-frp-panel/internal/comm"
	"github.com/xxl6097/go-frp-panel/pkg/utils"
	"github.com/xxl6097/go-service/gservice/ukey"
	utils2 "github.com/xxl6097/go-service/gservice/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (this *frps) userHandlers(helper *httppkg.RouterRegisterHelper) {
	subRouter := helper.Router.NewRoute().Name("user").Subrouter()
	subRouter.Use(helper.AuthMiddleware)
	// apis
	subRouter.HandleFunc("/api/token/add", this.apiUserCreate).Methods("POST")
	subRouter.HandleFunc("/api/token/del", this.apiUserDelete).Methods("POST")
	subRouter.HandleFunc("/api/token/chg", this.apiUserUpdate).Methods("POST")
	subRouter.HandleFunc("/api/token/all", this.apiUserAll).Methods("GET")

	subRouter.HandleFunc("/api/client/get", this.apiClientGet).Methods("GET")
	subRouter.HandleFunc("/api/client/gen", this.apiClientGen).Methods("POST")
	subRouter.HandleFunc("/api/client/toml", this.apiClientToml).Methods("POST")
	subRouter.HandleFunc("/api/client/user/import", this.apiClientUserImport).Methods("POST")
	subRouter.HandleFunc("/api/client/user/export", this.apiClientUserExport).Methods("POST")
	subRouter.HandleFunc("/api/client/upload", this.apiClientUpload).Methods("POST")

}

func (this *frps) apiUserCreate(w http.ResponseWriter, r *http.Request) {
	res, f := comm.Response(r)
	defer f(w)
	//body, err := io.ReadAll(r.Body)
	//if err != nil {
	//	res.Response(400, fmt.Sprintf("read request body error: %v", err))
	//	glog.Warnf("%s", res.Msg)
	//	return
	//}
	//fmt.Println(string(body))
	u, err := utils.GetDataByJson[User](r)
	if err != nil {
		res.Err(err)
		glog.Errorf("%v decode user err: %+v", err, u)
		return
	}
	if u == nil {
		res.Error("token is nil")
		return
	}
	err = u.CreateUser()
	if err != nil {
		res.Err(err)
		glog.Errorf("%v create user err: %+v", err, u)
		return
	}
	res.Ok("密钥创建成功")
}

func (this *frps) apiUserDelete(w http.ResponseWriter, r *http.Request) {
	res, f := comm.Response(r)
	defer f(w)
	users, err := utils.GetDataByJson[[]struct {
		User string `json:"user"`
	}](r)
	if err != nil {
		res.Err(err)
		glog.Error(err)
		return
	}
	if users == nil {
		res.Error("tokens is nil")
		return
	}
	for _, u := range *users {
		err = DeleteUser(u.User)
	}
	//err = this.repo.Delete(u.User)
	//if err != nil {
	//	res.Err(err)
	//	return
	//}
	res.Ok("密钥删除成功")
}

func (this *frps) apiUserUpdate(w http.ResponseWriter, r *http.Request) {
	res, f := comm.Response(r)
	defer f(w)
	u, err := utils.GetDataByJson[User](r)
	if err != nil {
		res.Err(err)
		return
	}
	if u == nil {
		res.Error("token is nil")
		return
	}
	glog.Printf("%+v\n", u)
	//userFilePath := filepath.Join(this.workDir, "user", fmt.Sprintf("%s.json", u.User))
	//if gore.FileExists(userFilePath) {
	//	os.Remove(userFilePath)
	//}
	//err = u.CreateUser(userFilePath)
	err = u.UpdateUser()

	if err != nil {
		res.Err(err)
		glog.Error(err)
		return
	}
	res.Ok("密钥更新成功")
	a, _ := GetUserAll()
	fmt.Printf("结果：%+v\n", a)
}
func (this *frps) apiUserAll(w http.ResponseWriter, r *http.Request) {
	res, f := comm.Response(r)
	defer f(w)
	datas, err := GetUserAll()
	if err != nil {
		res.Error("无数据")
		glog.Error(err)
		return
	}
	res.Sucess("全部数据获取成功", datas)
	glog.Infof("%+v\n", datas)
}

func (this *frps) apiClientGet(w http.ResponseWriter, r *http.Request) {
	res, f := comm.Response(r)
	defer f(w)
	binPath, err := os.Executable()
	if err != nil {
		res.Error(fmt.Sprintf("获取当前可执行文件路径出错: %v\n", err))
		glog.Error(res.Msg)
		return
	}
	configDir := filepath.Dir(binPath)
	configPath := filepath.Join(configDir, "clients")
	glog.Infof("扫描路径:%s", configPath)
	res.Data = utils.GetNodes(configPath)
	glog.Infof("扫描结果:%v", res.Data)
}

func (this *frps) parseUser(data map[string]interface{}) {
	glog.Println(data)
	u := User{
		User:       data["user"].(string),
		Token:      data["token"].(string),
		Comment:    data["comment"].(string),
		Ports:      ToPorts(data["ports"].([]any)),
		Domains:    data["domains"].([]string),
		Subdomains: data["subdomains"].([]string),
		Enable:     data["enable"].(bool),
	}
	glog.Error(u)
}

func (this *frps) apiClientGen(w http.ResponseWriter, r *http.Request) {
	//res := &comm.GeneralResponse{Code: 0}

	//body1, err := io.ReadAll(r.Body)
	//if err != nil {
	//	res.Response(400, fmt.Sprintf("read request body error: %v", err))
	//	glog.Warnf("%s", res.Msg)
	//	return
	//}
	//fmt.Println(string(body1))

	body, err := utils.GetDataByJson[struct {
		BinPath string `json:"binPath"`
		BinUrl  string `json:"binUrl"`
		Addr    string `json:"addr"`
		User    User   `json:"user"`
	}](r)
	if err != nil {
		glog.Error("解析Json对象失败", err)
		return
	}
	if body == nil {
		msg := "json对象nil"
		glog.Error(msg)
		http.Error(w, "json对象nil", http.StatusInternalServerError)
		return
	}
	glog.Debugf("body:%+v\n", body)
	if utils2.IsURL(body.BinUrl) {
		dstPath, err := utils.DownLoad(body.BinUrl)
		if err != nil {
			msg := fmt.Errorf("下载文件失败～%v", err)
			glog.Error(msg)
			http.Error(w, msg.Error(), http.StatusNotImplemented)
			return
		}
		body.BinPath = dstPath
	}
	if body.User.User == "" {
		msg := fmt.Errorf("用户名空")
		glog.Error(msg)
		http.Error(w, msg.Error(), http.StatusBadGateway)
		return
	}
	binPath := body.BinPath
	if binPath == "" {
		msg := fmt.Errorf("bin文件路径空")
		glog.Error(msg)
		http.Error(w, msg.Error(), http.StatusServiceUnavailable)
		return
	}
	glog.Infof("binPath: %s %+v\n", binPath, body)
	tpl, err := os.Open(binPath)
	if err != nil {
		msg := fmt.Errorf("打开文件失败：%v", err)
		glog.Error(msg)
		http.Error(w, msg.Error(), http.StatusGatewayTimeout)
		return
	}
	defer tpl.Close()

	w.Header().Add("Content-Transfer-Encoding", "binary")
	w.Header().Add("Content-Type", "application/octet-stream")
	if stat, err := tpl.Stat(); err == nil {
		w.Header().Add(`Content-Length`, strconv.FormatInt(stat.Size(), 10))
	}
	w.Header().Add(`Content-Disposition`, fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(binPath)))
	//cfgBuffer := ukey.GetBuffer()
	cfgBuffer := bytes.Repeat([]byte{byte(ukey.B)}, len(ukey.GetBuffer()))
	cfg := comm.BufferConfig{
		Addr:       body.Addr,
		Port:       GetCfgModel().Frps.BindPort,
		User:       body.User.User,
		Token:      body.User.Token,
		Comment:    body.User.Comment,
		Ports:      body.User.Ports,
		Domains:    body.User.Domains,
		Subdomains: body.User.Subdomains,
	}

	cfgNewBytes, err := ukey.GenConfig(cfg, false)
	if err != nil {
		msg := fmt.Errorf("文件签名失败：%v", err)
		glog.Error(msg)
		http.Error(w, msg.Error(), http.StatusHTTPVersionNotSupported)
		return
	}
	prevBuffer := make([]byte, 0)
	for {
		thisBuffer := make([]byte, 1024)
		n, err := tpl.Read(thisBuffer)
		thisBuffer = thisBuffer[:n]
		tempBuffer := append(prevBuffer, thisBuffer...)
		bufIndex := bytes.Index(tempBuffer, cfgBuffer)
		if bufIndex > -1 {
			tempBuffer = bytes.Replace(tempBuffer, cfgBuffer, cfgNewBytes, -1)
		}
		w.Write(tempBuffer[:len(prevBuffer)])
		prevBuffer = tempBuffer[len(prevBuffer):]
		if err != nil {
			break
		}
	}
	if len(prevBuffer) > 0 {
		w.Write(prevBuffer)
		prevBuffer = nil
	}
}

func (this *frps) apiClientUserExport(w http.ResponseWriter, r *http.Request) {
	res := &comm.GeneralResponse{Code: 0}
	binpath, err := os.Executable()
	if err != nil {
		res.Err(err)
		glog.Error(err)
		return
	}
	fileName := fmt.Sprintf("user_%s.zip", utils.GetFileNameByTime())
	userDir := filepath.Join(filepath.Dir(binpath), "user")
	zipFilePath := filepath.Join(filepath.Dir(binpath), fileName)
	err = utils.Zip(userDir, zipFilePath)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		glog.Error("GetDataByJson", err)
		return
	}

	tpl, err := os.Open(zipFilePath)
	if err != nil {
		res.Err(fmt.Errorf("打开文件失败：%v", err))
		bb, err := json.Marshal(res)
		if err != nil {
			glog.Errorf("marshal result error: %v", err)
			w.WriteHeader(400)
			return
		}
		w.Write(bb)
		return
	}
	defer tpl.Close()

	w.Header().Add("Content-Transfer-Encoding", "binary")
	w.Header().Add("Content-Type", "application/octet-stream")
	if stat, err := tpl.Stat(); err == nil {
		w.Header().Add(`Content-Length`, strconv.FormatInt(stat.Size(), 10))
	}
	w.Header().Add(`Content-Disposition`, fmt.Sprintf("attachment; filename=\"%s\"", fileName))

	prevBuffer := make([]byte, 0)
	for {
		thisBuffer := make([]byte, 1024)
		n, err := tpl.Read(thisBuffer)
		thisBuffer = thisBuffer[:n]
		tempBuffer := append(prevBuffer, thisBuffer...)
		w.Write(tempBuffer[:len(prevBuffer)])
		prevBuffer = tempBuffer[len(prevBuffer):]
		if err != nil {
			break
		}
	}
	if len(prevBuffer) > 0 {
		w.Write(prevBuffer)
		prevBuffer = nil
	}
}

func (this *frps) apiClientUserImport(w http.ResponseWriter, r *http.Request) {
	res := &comm.GeneralResponse{Code: 0}
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		res.Error("body can't be empty")
		glog.Error(res.Msg)
		return
	}
	// 获取上传的文件
	file, handler, err := r.FormFile("file")
	if err != nil {
		res.Error("body no file")
		return
	}
	defer file.Close()

	binpath, err := os.Executable()
	if err != nil {
		res.Err(err)
		glog.Error(binpath, err)
		return
	}

	userDir := filepath.Join(filepath.Dir(binpath), "user")

	if err = utils.DirCheck(userDir); err != nil {
		res.Err(fmt.Errorf("check config dir err: %v", err))
		glog.Error(res.Msg)
		return
	}
	glog.Info(handler.Filename)
	ext := strings.ToLower(filepath.Ext(handler.Filename)) // 统一转为小写
	switch ext {
	case ".zip":
		dstFilePath := filepath.Join(os.TempDir(), handler.Filename)
		dst, err := os.Create(dstFilePath)
		if err != nil {
			res.Error(fmt.Sprintf("create file %s error: %v", handler.Filename, err))
			return
		}
		buf := this.upgrade.GetBuffer().Get().([]byte)
		defer this.upgrade.GetBuffer().Put(buf)
		_, err = io.CopyBuffer(dst, file, buf)
		dst.Close()

		//err = utils.SaveFile(file, handler.Size, dstFilePath)
		if err != nil {
			res.Error(err.Error())
			return
		}
		err = utils.UnzipToRoot(dstFilePath, userDir, true)
		if err == nil {
			utils.Delete(dstFilePath, "用户文件")
			glog.Info("解压成功", userDir)
		}
		break
	case ".json":
		dstFilePath := filepath.Join(userDir, handler.Filename)
		dst, err := os.Create(dstFilePath)
		if err != nil {
			res.Error(fmt.Sprintf("create file %s error: %v", handler.Filename, err))
			return
		}
		buf := this.upgrade.GetBuffer().Get().([]byte)
		defer this.upgrade.GetBuffer().Put(buf)
		_, err = io.CopyBuffer(dst, file, buf)
		dst.Close()
		//err = utils.SaveFile(file, handler.Size, dstFilePath)
		if err != nil {
			res.Error(err.Error())
			return
		}
		glog.Info("导入成功", dstFilePath)
		break
	}
}

func (this *frps) apiClientToml(w http.ResponseWriter, r *http.Request) {
	res := &comm.GeneralResponse{Code: 0}

	body, err := utils.GetDataByJson[struct {
		BinPath string `json:"binPath"`
		BinUrl  string `json:"binUrl"`
		Addr    string `json:"addr"`
		User    User   `json:"user"`
	}](r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		glog.Error("GetDataByJson", err)
		return
	}
	if body == nil {
		res.Err(errors.New("body is nil"))
		w.WriteHeader(http.StatusInternalServerError)
		glog.Error("body is nil")
		return
	}
	if body.BinUrl != "" && utils2.IsURL(body.BinUrl) {
		dstPath, err1 := utils.DownLoad(body.BinUrl)
		if err1 == nil {
			body.BinPath = dstPath
		}
	}

	fileName := fmt.Sprintf("%s_frpc.toml", body.User.User)

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("serverAddr = \"%s\"\n", body.Addr))
	sb.WriteString(fmt.Sprintf("serverPort = %d\n", GetCfgModel().Frps.BindPort))
	sb.WriteString(fmt.Sprintf("user = \"%s\"\n", body.User.User))
	sb.WriteString(fmt.Sprintf("metadatas.token = \"%s\"\n", body.User.Token))
	size := sb.Len()

	w.Header().Add("Content-Transfer-Encoding", "binary")
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add(`Content-Length`, strconv.Itoa(size))
	w.Header().Add(`Content-Disposition`, fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Write([]byte(sb.String()))
}

func (this *frps) apiClientUpload(w http.ResponseWriter, r *http.Request) {
	res, f := comm.Response(r)
	defer f(w)
	//err := r.ParseMultipartForm(32 << 20)
	//if err != nil {
	//	res.Error("body can't be empty")
	//	glog.Error(res.Msg)
	//	return
	//}
	// 获取上传的文件
	file, handler, err := r.FormFile("file")
	if err != nil {
		res.Error("body no file")
		return
	}
	defer file.Close()
	binPath, err := os.Executable()
	if err != nil {
		res.Error(fmt.Sprintf("获取当前可执行文件路径出错: %v\n", err))
		glog.Error(res.Msg)
		return
	}
	binDir := filepath.Dir(binPath)
	clientsDir := filepath.Join(binDir, "clients")
	err = utils.EnsureDir(clientsDir)
	if err != nil {
		res.Error(fmt.Sprintf("文件夹创建失败: %v\n", err))
		glog.Error(res.Msg)
		return
	}

	dstFilePath := filepath.Join(clientsDir, handler.Filename)
	//dstFilePath 名称为上传文件的原始名称
	dst, err := os.Create(dstFilePath)
	if err != nil {
		res.Error(fmt.Sprintf("create file %s error: %v", handler.Filename, err))
		return
	}
	buf := this.upgrade.GetBuffer().Get().([]byte)
	defer this.upgrade.GetBuffer().Put(buf)
	_, err = io.CopyBuffer(dst, file, buf)
	dst.Close()
	if err != nil {
		res.Error(err.Error())
		glog.Error(res.Msg)
		return
	}
	glog.Println("客户端路径", clientsDir)
	glog.Println("文件上传成功", dstFilePath)
	err = utils.UnzipToRoot(dstFilePath, clientsDir, true)
	if err != nil {
		res.Error(err.Error())
		glog.Error(res.Msg)
		return
	} else {
		utils.Delete(dstFilePath)
	}
	res.Ok("文件上传成功～")
}
