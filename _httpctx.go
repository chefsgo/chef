package chef

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"
)

type (
	HttpFunc func(*Access)
	Access   struct {
		*context

		module *httpModule

		index int        //下一个索引
		nexts []HttpFunc //方法列表

		thread   HttpThread
		request  *http.Request
		response http.ResponseWriter

		//是否颁发token
		//issue=true时，需要cookie或是data返回token
		issue bool

		charset string

		// id         string	//2022被token取代
		Name    string
		Config  Router
		Setting Map

		Site       string //站点key
		siteConfig SiteConfig

		Method string //请求方法，大写
		Host   string //请求域名
		Domain string
		Path   string //请求路径
		Uri    string //请求uri
		Ajax   bool

		headers        map[string]string
		cookies        map[string]http.Cookie
		sessions       map[string]Any
		sessionchanged bool

		Client Map //客户端信息
		Params Map //uri 中的参数
		Query  Map //querystring 参数
		Form   Map //表单参数
		Upload Map //上传文件数据
		Data   Map //viewdata传给视图

		Value Map //所有参数汇总
		Args  Map //定义的args解析后的参数
		Sign  Map //sign会话校验对象，支持查询数据库
		Item  Map //查询单个数据库对象
		Local Map //上下文传递数据

		Code int    //返回HTTP状态
		Type string //返回内容类型
		Body Any    //返回body

		Url *httpUrl
	}
)

func httpEmpty(http *httpModule) *Access {
	return &Access{
		context: newcontext("httpEmpty"),
		module:  http,
	}
}
func httpContext(module *httpModule, thread HttpThread) *Access {
	ctx := &Access{
		context: newcontext(), module: module,
		index: 0, nexts: make([]HttpFunc, 0), charset: UTF8,
		thread: thread, request: thread.Request(), response: thread.Response(),
		Setting: make(Map),
		headers: make(map[string]string), cookies: make(map[string]http.Cookie), sessions: make(Map),
		Client: make(Map), Params: make(Map), Query: make(Map), Form: make(Map), Upload: make(Map), Data: make(Map),
		Value: make(Map), Args: make(Map), Sign: make(Map), Item: make(Map), Local: make(Map),
	}

	ctx.Name = thread.Name()
	ctx.Site = thread.Site()
	ctx.Params = thread.Params()

	ctx.Method = strings.ToUpper(ctx.request.Method)
	ctx.Uri = ctx.request.RequestURI
	ctx.Path = ctx.request.URL.Path

	//使用域名去找site
	ctx.Host = ctx.request.Host
	if strings.Contains(ctx.Host, ":") {
		hosts := strings.Split(ctx.Host, ":")
		if len(hosts) > 0 {
			ctx.Host = hosts[0]
		}
	}

	//获取获名对应的 site.key
	if ctx.Site == "" {
		if site, ok := module.hosts[ctx.Host]; ok {
			ctx.Site = site
		}
	}
	//获取site的配置
	if vvvv, ookk := module.sites[ctx.Site]; ookk {
		ctx.siteConfig = vvvv
	} else {
		ctx.siteConfig = SiteConfig{}
	}

	// 获取根域名，如果IP直接访问，这里会有问题
	// 所以需要先判断，是不是直接IP访问，不是IP才解析根域
	ip := net.ParseIP(ctx.Host)
	if ip == nil {
		parts := strings.Split(ctx.Host, ".")
		if len(parts) >= 2 {
			l := len(parts)
			ctx.Domain = parts[l-2] + "." + parts[l-1]
		}
	}

	ctx.Url = &httpUrl{ctx}

	return ctx
}

func (ctx *Access) clear() {
	ctx.index = 0
	ctx.nexts = make([]HttpFunc, 0)
}
func (ctx *Access) next(nexts ...HttpFunc) {
	ctx.nexts = append(ctx.nexts, nexts...)
}

func (ctx *Access) Next() {
	if len(ctx.nexts) > ctx.index {
		next := ctx.nexts[ctx.index]
		ctx.index++
		if next != nil {
			next(ctx)
		} else {
			ctx.Next()
		}
	} else {
		//是否需要做执行完的处理
	}
}

func (ctx *Access) Charset(charsets ...string) string {
	if ctx == nil {
		return UTF8
	}
	if len(charsets) > 0 && charsets[0] != "" {
		ctx.charset = charsets[0]
	}
	return ctx.charset
}

func (ctx *Access) sessional(defs ...bool) bool {
	sessional := false
	if len(defs) > 0 {
		sessional = defs[0]
	}

	if vv, ok := ctx.Setting["session"].(bool); ok {
		sessional = vv
	}

	//如果有auth节，强制使用session
	if ctx.Config.Sign != nil {
		sessional = true
	}

	//如果SESSION已经被更新
	if ctx.sessionchanged {
		sessional = true
	}

	return sessional
}

//客户端请求校验
//接口请求校验
//设备，系统，版本，客户端，版本号，时间戳，签名
//{device}/{system}/{version}/{client}/{number}/{time}/{sign}
func (ctx *Access) clientHandler() Res {

	checking := false
	if ctx.siteConfig.Validate != "" {
		checking = true
	}

	//个别路由通行
	if vv, ok := ctx.Setting["passport"].(bool); ok && vv {
		checking = false
	}
	if vv, ok := ctx.Setting["validate"].(bool); ok {
		checking = vv
	}
	//待完善
	// if vv := ctx.Header("Debug"); vv == Secret {
	if vv := ctx.Header("Debug"); vv != "" {
		checking = false //调试通行证
	}

	cs := ""
	if vv := ctx.Header("Client"); vv != "" {
		cs = strings.TrimSpace(vv)
	}

	if cs == "" {
		if checking {
			return Invalid
		} else {
			return OK
		}
	}

	//eses := ark.Codec.Decrypt(cs)

	args := Vars{
		"client": Var{Type: "string", Required: true, Decode: ctx.siteConfig.Validate},
	}
	data := Map{
		"client": cs,
	}
	value := Map{}

	res := mBasic.Mapping(args, data, value, false, false, ctx.context)

	if res != nil && res.Fail() {
		return Invalid
	}

	client := value["client"].(string)

	vals := strings.Split(client, "/")
	if len(vals) < 7 {
		//Debug("client", "Length", err, client)
		if checking {
			return Invalid
		}
		return OK
	}

	//保存参数
	ctx.Client["device"] = vals[0]
	ctx.Client["system"] = vals[1]
	ctx.Client["version"] = vals[2]
	ctx.Client["client"] = vals[3]
	ctx.Client["number"] = vals[4]
	ctx.Client["time"] = vals[5]
	ctx.Client["sign"] = vals[6]

	//实际传的，path不需要传，是传的签名
	format := `{device}/{system}/{version}/{client}/{number}/{time}/{path}`
	if ctx.siteConfig.Format != "" {
		format = ctx.siteConfig.Format
	}

	format = strings.Replace(format, "{device}", vals[0], -1)
	format = strings.Replace(format, "{system}", vals[1], -1)
	format = strings.Replace(format, "{version}", vals[2], -1)
	format = strings.Replace(format, "{client}", vals[3], -1)
	format = strings.Replace(format, "{number}", vals[4], -1)
	format = strings.Replace(format, "{time}", vals[5], -1)
	format = strings.Replace(format, "{path}", ctx.Path, -1)

	sign := strings.ToLower(util.Md5(format))

	//Debug("vvv", sign, format, value)

	if sign != vals[6] && checking {
		//Debug("ClientSign", ctx.Uri, sign, data["client"], value["client"])
		return Invalid
	}

	//时间对比
	if ctx.siteConfig.Timeout != "" {
		d, e := util.ParseDuration(ctx.siteConfig.Timeout)
		if e != nil {
			//设置的超时格式有问题
			d = time.Minute * 5
		}
		now := time.Now()
		if vvd, err := strconv.ParseInt(vals[5], 10, 64); err != nil {
			//时间有问题
			return Invalid
		} else {
			tms := time.Unix(vvd, 0).Add(d)
			if tms.Unix() < now.Unix() {
				//失败时间比当前时间小，失败
				return Invalid
			}
		}

	}

	//到这里才成功
	return OK
}

//专门处理base64格式的文件上传
func (ctx *Access) formUploadHandler(values []string) []Map {
	files := []Map{}

	baseExp := regexp.MustCompile(`data\:(.*)\;base64,(.*)`)
	for _, base := range values {
		arr := baseExp.FindStringSubmatch(base)
		if len(arr) == 3 {
			baseBytes, err := base64.StdEncoding.DecodeString(arr[2])
			if err == nil {
				h := sha1.New()
				if _, err := h.Write(baseBytes); err == nil {
					hash := fmt.Sprintf("%x", h.Sum(nil))

					mimeType := arr[1]
					extension := mBasic.Extension(mimeType)
					filename := fmt.Sprintf("%s.%s", hash, extension)
					length := len(baseBytes)

					tempfile := "up_*"
					if extension != "" {
						tempfile = fmt.Sprintf("%s.%s", tempfile, extension)
					}

					file, err := ctx.TempFile(tempfile)
					if err == nil {
						defer file.Close()
						if _, err := file.Write(baseBytes); err == nil {
							files = append(files, Map{
								"hash": hash,
								"name": filename,
								"type": strings.ToLower(extension),
								"mime": mimeType,
								"size": length,
								"file": file.Name(),
							})
						}
					}

				}
			}
		}
	}

	return files
}

func (ctx *Access) formHandler() Res {
	var req = ctx.request

	//URL中的参数
	for k, v := range ctx.Params {
		ctx.Value[k] = v
	}

	//urlquery
	for k, v := range req.URL.Query() {
		if len(v) == 1 {
			ctx.Query[k] = v[0]
			ctx.Value[k] = v[0]
		} else if len(v) > 1 {
			ctx.Query[k] = v
			ctx.Value[k] = v
		}
	}

	//是否AJAX请求，可能在拦截器里手动指定为true了，就不处理了
	if ctx.Ajax == false {
		if ctx.Header("X-Requested-With") != "" {
			ctx.Ajax = true
		} else if ctx.Header("Ajax") != "" {
			ctx.Ajax = true
		} else {
			ctx.Ajax = false
		}
	}

	//客户端的默认语言
	if al := ctx.Header("Accept-Language"); al != "" {
		accepts := strings.Split(al, ",")
		if len(accepts) > 0 {
		llll:
			for _, accept := range accepts {
				if i := strings.Index(accept, ";"); i > 0 {
					accept = accept[0:i]
				}
				//遍历匹配
				for lang, config := range mBasic.langConfigs {
					for _, acccc := range config.Accepts {
						if strings.ToLower(acccc) == strings.ToLower(accept) {
							ctx.Lang(lang)
							break llll
						}
					}
				}
			}
		}
	}

	uploads := map[string][]Map{}

	//if ctx.Method == "POST" || ctx.Method == "PUT" || ctx.Method == "DELETE" || ctx.Method == "PATCH" {
	if ctx.Method != "GET" {
		//根据content-type来处理
		ctype := ctx.Header("Content-Type")
		if strings.Contains(ctype, "json") {
			body, err := ioutil.ReadAll(req.Body)
			if err == nil {
				ctx.Body = RawBody(body)

				m := Map{}
				err := JSONUnmarshal(body, &m)
				if err == nil {
					//遍历JSON对象
					for k, v := range m {
						ctx.Form[k] = v
						ctx.Value[k] = v

						if vs, ok := v.(string); ok {
							baseFiles := ctx.formUploadHandler([]string{vs})
							if len(baseFiles) > 0 {
								uploads[k] = baseFiles
							}
						} else if vs, ok := v.([]Any); ok {
							vsList := []string{}
							for _, vsa := range vs {
								if vss, ok := vsa.(string); ok {
									vsList = append(vsList, vss)
								}
							}

							if len(vsList) > 0 {
								baseFiles := ctx.formUploadHandler(vsList)
								if len(baseFiles) > 0 {
									uploads[k] = baseFiles
								}
							}
						}

					}
				}
			}
		} else if strings.Contains(ctype, "xml") {
			body, err := ioutil.ReadAll(req.Body)
			if err == nil {
				ctx.Body = RawBody(body)

				m := Map{}
				err := xml.Unmarshal(body, &m)
				if err == nil {
					//遍历XML对象
					for k, v := range m {
						ctx.Form[k] = v
						ctx.Value[k] = v
					}
				}
			}
		} else {

			// if ctype=="application/x-www-form-urlencoded" || ctype=="multipart/form-data" {
			// }

			err := req.ParseMultipartForm(32 << 20)
			if err != nil {
				//表单解析有问题，就处理成原始STRING
				body, err := ioutil.ReadAll(req.Body)
				if err == nil {
					ctx.Body = RawBody(body)
				}

			}

			names := []string{}
			values := url.Values{}
			// uploads := map[string][]Map{}

			if req.MultipartForm != nil {

				//处理表单，这里是否应该直接写入ctx.Form比较好？
				for k, v := range req.MultipartForm.Value {
					//有个问题，当type=file时候，又不选文件的时候，value里会存在一个空字串的value
					//如果同一个form name 有多条记录，这时候会变成一个[]string，的空串数组
					//这时候，mapping解析文件的时候[file]就会出问题，会判断文件类型，这时候是[]string就出问题了
					// ctx.Form[k] = v
					names = append(names, k)
					values[k] = v
				}

				//FILE可能要弄成JSON，文件保存后，MIME相关的东西，都要自己处理一下
				for k, v := range req.MultipartForm.File {
					//这里应该保存为数组
					files := []Map{}

					//处理多个文件
					for _, f := range v {

						if f.Size <= 0 || f.Filename == "" {
							continue
						}

						hash := ""
						filename := f.Filename
						mimetype := f.Header.Get("Content-Type")
						extension := strings.ToLower(path.Ext(filename))
						if extension != "" {
							extension = extension[1:] //去掉点.
						}

						var length int64 = f.Size

						//先计算hash
						if file, err := f.Open(); err == nil {

							h := sha1.New()
							if _, err := io.Copy(h, file); err == nil {
								hash = fmt.Sprintf("%x", h.Sum(nil))

								//重新定位
								file.Seek(0, 0)

								tempfile := "fs_*"
								if extension != "" {
									tempfile = fmt.Sprintf("%s.%s", tempfile, extension)
								}

								tempFile, err := ctx.TempFile(tempfile)
								if err == nil {
									io.Copy(tempFile, file) //保存文件
									tempFile.Close()

									msg := Map{
										"hash": hash,
										"name": filename,
										"type": extension,
										"mime": mimetype,
										"size": length,
										"file": tempFile.Name(),
									}

									files = append(files, msg)
								}

							}

							//最后关闭文件
							file.Close()
						}

						uploads[k] = files
					}
				}

			} else if req.PostForm != nil {
				for k, v := range req.PostForm {
					names = append(names, k)
					values[k] = v
				}

			} else if req.Form != nil {
				for k, v := range req.Form {
					names = append(names, k)
					values[k] = v
				}
			}

			tomlroot := map[string][]string{}
			tomldata := map[string]map[string][]string{}

			//顺序很重要
			tomlexist := map[string]bool{}
			tomlnames := []string{}

			//统一解析
			for _, k := range names {
				v := values[k]

				//写入form
				if len(v) == 1 {
					ctx.Form[k] = v[0]
				} else if len(v) > 1 {
					ctx.Form[k] = v
				}

				//解析base64文件 begin
				baseFiles := ctx.formUploadHandler([]string(v))
				if len(baseFiles) > 0 {
					uploads[k] = baseFiles
				}
				//解析base64文件 end

				// key := fmt.Sprintf("value[%s]", k)
				// forms[k] = v

				if strings.Contains(k, ".") {

					//以最后一个.分割，前为key，后为field
					i := strings.LastIndex(k, ".")
					key := k[:i]
					field := k[i+1:]

					if vv, ok := tomldata[key]; ok {
						vv[field] = v
					} else {
						tomldata[key] = map[string][]string{
							field: v,
						}
					}

					if _, ok := tomlexist[key]; ok == false {
						tomlexist[key] = true
						tomlnames = append(tomlnames, key)
					}

				} else {
					tomlroot[k] = v
				}

				//这里不写入， 解析完了才
				// ctx.Value[k] = v
			}

			lines := []string{}
			for kk, vv := range tomlroot {
				if len(vv) > 1 {
					lines = append(lines, fmt.Sprintf(`%s = ["""%s"""]`, kk, strings.Join(vv, `""","""`)))
				} else {
					lines = append(lines, fmt.Sprintf(`%s = """%s"""`, kk, vv[0]))
				}
			}
			for _, kk := range tomlnames {
				vv := tomldata[kk]

				//普通版
				// lines = append(lines, fmt.Sprintf("[%s]", kk))
				// for ff,fv := range vv {
				// 	if len(fv) > 1 {
				// 		lines = append(lines, fmt.Sprintf(`%s = ["%s"]`, ff, strings.Join(fv, `","`)))
				// 	} else {
				// 		lines = append(lines, fmt.Sprintf(`%s = "%s"`, ff, fv[0]))
				// 	}
				// }

				//数组版
				//先判断一下，是不是map数组
				length := 0
				for _, fv := range vv {
					if length == 0 {
						length = len(fv)
					} else {
						if length != len(fv) {
							length = -1
							break
						}
					}
				}

				//如果length>1是数组，并且相等
				if length > 1 {
					for i := 0; i < length; i++ {
						lines = append(lines, fmt.Sprintf("[[%s]]", kk))
						for ff, fv := range vv {
							lines = append(lines, fmt.Sprintf(`%s = """%s"""`, ff, fv[i]))
						}
					}

				} else {
					lines = append(lines, fmt.Sprintf("[%s]", kk))
					for ff, fv := range vv {
						if len(fv) > 1 {
							lines = append(lines, fmt.Sprintf(`%s = ["""%s"""]`, ff, strings.Join(fv, `""","""`)))
						} else {
							lines = append(lines, fmt.Sprintf(`%s = """%s"""`, ff, fv[0]))
						}
					}
				}
			}

			value := parseTOML(strings.Join(lines, "\n"))
			if value != nil {
				for k, v := range value {
					ctx.Value[k] = v
				}
			} else {
				for k, v := range values {
					if len(v) == 1 {
						ctx.Value[k] = v[0]
					} else if len(v) > 1 {
						ctx.Value[k] = v
					}
				}
			}
		}
	}

	for k, v := range uploads {
		if len(v) == 1 {
			ctx.Value[k] = v[0]
			ctx.Upload[k] = v[0]
		} else if len(v) > 1 {
			ctx.Value[k] = v
			ctx.Upload[k] = v
		}
	}

	return OK
}

//处理参数
func (ctx *Access) argsHandler() Res {

	if ctx.Config.Args != nil {

		argsValue := Map{}
		res := mBasic.Mapping(ctx.Config.Args, ctx.Value, argsValue, ctx.Config.Nullable, false, ctx.context)
		if res != nil && res.Fail() {
			return res
		}

		for k, v := range argsValue {
			ctx.Args[k] = v
		}
	}

	return OK
}

//处理认证
func (ctx *Access) authHandler() Res {
	if ctx.Config.Token && ctx.token == "" {
		return Unauthorized
	}
	if ctx.Config.Auth && false == ctx.Authorized() {
		return Unauthorized
	}
	return OK
}

//处理认证
func (ctx *Access) signHandler() Res {

	if ctx.Config.Sign != nil {
		saveMap := Map{}

		for authKey, authConfig := range ctx.Config.Sign {
			ohNo := false

			authSign := authConfig.Sign
			if authSign == "" {
				authSign = authKey
			}

			//判断是否登录
			if ctx.Signed(authSign) {

				//改为使用方法调用
				if authConfig.Method != "" {
					args := "id"
					if authConfig.Args != "" {
						args = authConfig.Args
					}

					id := ctx.Signal(authSign)
					item := ctx.Invoke(authConfig.Method, Map{args: id})

					if item == nil {
						if authConfig.Required {
							if authConfig.Error != nil {
								return authConfig.Error
							} else {
								return textResult("_auth_error_" + authKey)
							}
						}
					} else {
						saveMap[authKey] = item
					}
				}

			} else {
				ohNo = true
			}

			//到这里是未登录的
			//而且是必须要登录，才显示错误
			if ohNo && authConfig.Required {
				if authConfig.Empty != nil {
					return authConfig.Empty
				} else {
					return textResult("_auth_empty_" + authKey)
				}
			}
		}

		//存入
		for k, v := range saveMap {
			ctx.Sign[k] = v
		}
	}

	return OK
}

//Entity实体处理
func (ctx *Access) itemHandler() Res {

	if ctx.Config.Find != nil {
		saveMap := Map{}

		for itemKey, config := range ctx.Config.Find {

			//itemName := itemKey
			//if vv,ok := config[kNAME].(string); ok && vv != "" {
			//	itemName = vv
			//}

			realKey := "id"
			if config.Value != "" {
				realKey = config.Value
			}
			var realVal Any
			if vv, ok := ctx.Args[realKey]; ok {
				realVal = vv
			} else if vv, ok := ctx.Value[realKey]; ok {
				realVal = vv
			}

			if realVal == nil && config.Required {
				if config.Empty != nil {
					return config.Empty
				} else {
					return textResult("_item_empty_" + itemKey)
				}
			} else {

				//判断是否需要查询数据
				if config.Method != "" && realVal != nil {
					args := "id"
					if config.Args != "" {
						args = config.Args
					}
					//要查询库
					item := ctx.Invoke(config.Method, Map{args: realVal})
					if item == nil && config.Required {
						if config.Error != nil {
							return config.Error
						} else {
							return textResult("_item_error_" + strings.Replace(itemKey, ".", "_", -1))
						}
					} else {
						saveMap[itemKey] = item
					}
				}

			}
		}

		//存入
		for k, v := range saveMap {
			ctx.Item[k] = v
		}
	}
	return OK
}

//接入错误处理流程，和模块挂钩了
func (ctx *Access) Found() {
	ctx.module.found(ctx)
}
func (ctx *Access) Erred(res Res) {
	ctx.lastError = res
	ctx.module.error(ctx)
}
func (ctx *Access) Failed(res Res) {
	ctx.lastError = res
	ctx.module.failed(ctx)
}
func (ctx *Access) Denied(res Res) {
	ctx.lastError = res
	ctx.module.denied(ctx)
}

//通用方法
func (ctx *Access) Header(key string, vals ...string) string {
	if len(vals) > 0 {
		ctx.headers[key] = vals[0]
		return vals[0]
	} else {
		//读header
		return ctx.request.Header.Get(key)
	}
}

//通用方法
func (ctx *Access) Cookie(key string, vals ...Any) string {
	//这个方法同步加密
	if len(vals) > 0 {
		vvv := vals[0]
		if vvv == nil {
			cookie := http.Cookie{Name: key, HttpOnly: true}
			cookie.MaxAge = -1
			ctx.cookies[key] = cookie
			return ""
		} else {
			switch val := vvv.(type) {
			case http.Cookie:
				// 改到 请求结束前，写入cookies时候统一加密
				val.HttpOnly = true
				ctx.cookies[key] = val
			case string:
				cookie := http.Cookie{Name: key, Value: val, Path: "/", HttpOnly: true}
				ctx.cookies[key] = cookie
			default:
				return ""
			}
		}
	} else {
		//读cookie
		c, e := ctx.request.Cookie(key)
		if e == nil {
			//这里是直接读的，所以要解密
			if vvv, err := TextDecrypt(c.Value); err == nil {
				return fmt.Sprintf("%v", vvv)
			}
			return c.Value
		}
	}
	return ""
}

func (ctx *Access) Session(key string, vals ...Any) Any {
	if len(vals) > 0 {
		ctx.sessionchanged = true
		val := vals[0]
		if val == nil {
			//删除SESSION
			delete(ctx.sessions, key)
		} else {
			//写入session
			ctx.sessions[key] = val
		}
		return val
	} else {
		return ctx.sessions[key]
	}
}

// //获取langString
// func (ctx *Access) String(key string, args ...Any) string {
// 	return mBasic.String(ctx.Lang(), key, args...)
// }

//----------------------- 签名系统 begin ---------------------------------
func (ctx *Access) signKey(key string) string {
	return fmt.Sprintf("$.sign.%s", key)
}
func (ctx *Access) Signed(key string) bool {
	key = ctx.signKey(key)
	if ctx.Session(key) != nil {
		return true
	}
	return false
}
func (ctx *Access) Signin(key string, id, name Any) {
	key = ctx.signKey(key)
	ctx.Session(key, Map{
		"id": fmt.Sprintf("%v", id), "name": fmt.Sprintf("%v", name),
	})
}
func (ctx *Access) Signout(key string) {
	key = ctx.signKey(key)
	ctx.Session(key, nil)
}
func (ctx *Access) Signal(key string) string {
	key = ctx.signKey(key)
	if vv, ok := ctx.Session(key).(Map); ok {
		if id, ok := vv["id"].(string); ok {
			return id
		}
	}
	return ""
}
func (ctx *Access) Signer(key string) string {
	key = ctx.signKey(key)
	if vv, ok := ctx.Session(key).(Map); ok {
		if id, ok := vv["name"].(string); ok {
			return id
		}
	}
	return ""
}

// //----------------------- 签名系统 end ---------------------------------

//直接远程反代文件，不下载到本地
func (ctx *Access) Remote(code string, names ...string) {
	//待处理，2022
	// coding := mFile.Decode(code)
	// var coding Any
	// if coding == nil {
	// 	ctx.Found()
	// 	return
	// }

	// //20211123，为什么apk要本地缓存？
	// //20220215是不是因为远程的http header问题，忘记了
	// if coding.Type() == "apk" {
	// 	ctx.Download(code, names...)
	// } else {

	// 	name := ""
	// 	if len(names) > 0 {
	// 		name = names[0]
	// 	}
	// 	url := Browse(code, name)
	// 	if url == "" {
	// 		ctx.Found()
	// 	} else {
	// 		ctx.Proxy(url)
	// 	}
	// }
}

//把文件拉到本地，然后再返回下载
func (ctx *Access) Download(code string, names ...string) {
	//待处理，2022
	// coding := mFile.Decode(code)
	// if coding == nil {
	// 	ctx.Found()
	// 	return
	// }

	// file, err := mFile.Download(coding)
	// if err != nil {
	// 	ctx.Found()
	// 	return
	// }

	// if len(names) > 0 {
	// 	//自动加扩展名
	// 	if coding.Type() != "" && !strings.HasSuffix(names[0], coding.Type()) {
	// 		names[0] += "." + coding.Type()
	// 	}
	// } else {
	// 	//未指定下载名的话，除了图片，全部自动加文件名
	// 	if !(coding.isimage() || coding.isvideo() || coding.isaudio()) {
	// 		names = append(names, coding.Hash()+"."+coding.Type())
	// 	}
	// }

	// ctx.File(file, coding.Type(), names...)
}

//生成并返回缩略图
func (ctx *Access) Thumbnail(code string, width, height, tttt int64) {
	//待处理2022
	// file, data, err := mFile.thumbnail(code, width, height, tttt)
	// if err != nil {
	// 	//ctx.Erred(errResult(err))
	// 	ctx.File(path.Join(ctx.module.config.Static, "shared", "nothing.png"), "png")
	// } else {
	// 	ctx.File(file, data.Type())
	// }
}

func (ctx *Access) Goto(url string) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	ctx.Body = httpGotoBody{url}
}
func (ctx *Access) Goback() {
	url := ctx.Url.Back()
	ctx.Goto(url)
}
func (ctx *Access) Text(text Any, codes ...int) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	real := ""
	if res, ok := text.(Res); ok {
		real = ctx.String(res.Text(), res.Args()...)
	} else if vv, ok := text.(string); ok {
		real = vv
	} else {
		real = fmt.Sprintf("%v", text)
	}

	//if len(types) > 0 {
	//	ctx.Type = types[0]
	//} else {
	//	ctx.Type = "text"
	//}

	if len(codes) > 0 {
		ctx.Code = codes[0]
	}
	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	ctx.Type = "text"
	ctx.Body = httpTextBody{real}
}
func (ctx *Access) Html(html string, codes ...int) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	if len(codes) > 0 {
		ctx.Code = codes[0]
	}
	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	ctx.Type = "html"
	ctx.Body = httpHtmlBody{html}
}
func (ctx *Access) Script(script string, codes ...int) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	if len(codes) > 0 {
		ctx.Code = codes[0]
	}
	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	ctx.Type = "script"
	ctx.Body = httpScriptBody{script}
}
func (ctx *Access) Json(json Any, codes ...int) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	if len(codes) > 0 {
		ctx.Code = codes[0]
	}
	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	ctx.Type = "json"
	ctx.Body = httpJsonBody{json}
}
func (ctx *Access) Jsonp(callback string, json Any, codes ...int) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	if len(codes) > 0 {
		ctx.Code = codes[0]
	}
	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	ctx.Type = "jsonp"
	ctx.Body = httpJsonpBody{json, callback}
}
func (ctx *Access) Xml(xml Any, codes ...int) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	if len(codes) > 0 {
		ctx.Code = codes[0]
	}
	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	ctx.Type = "xml"
	ctx.Body = httpXmlBody{xml}
}

func (ctx *Access) File(file string, mimeType string, names ...string) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	name := ""
	if len(names) > 0 {
		name = names[0]
	}
	if mimeType != "" {
		ctx.Type = mimeType
	} else {
		ctx.Type = "file"
	}

	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	ctx.Body = httpFileBody{file, name}
}

func (ctx *Access) Buffer(rd io.ReadCloser, mimeType string, names ...string) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	name := ""
	if len(names) > 0 {
		name = names[0]
	}

	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	if mimeType != "" {
		ctx.Type = mimeType
	} else {
		ctx.Type = "file"
	}
	ctx.Body = httpBufferBody{rd, name}
}
func (ctx *Access) Binary(bytes []byte, mimeType string, names ...string) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	if mimeType != "" {
		ctx.Type = mimeType
	} else {
		ctx.Type = "file"
	}
	name := ""
	if len(names) > 0 {
		name = names[0]
	}
	ctx.Body = httpDownBody{bytes, name}
}

func (ctx *Access) View(view string, types ...string) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	ctx.Type = "html"
	if len(types) > 0 {
		ctx.Type = types[0]
	}
	ctx.Body = httpViewBody{view, Map{}}
}

func (ctx *Access) Proxy(remote string) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	u, e := url.Parse(remote)
	if e != nil {
		ctx.Erred(errorResult(e))
	} else {
		ctx.Body = httpProxyBody{u}
	}
}

func (ctx *Access) Route(name string, values ...Map) {
	url := ctx.Url.Route(name, values...)
	ctx.Redirect(url)
}

func (ctx *Access) Redirect(url string) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	ctx.Goto(url)
}

func (ctx *Access) Alert(res Res, urls ...string) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	// code := mBasic.Code(res.Text(), res.Code())
	text := ctx.String(res.Text(), res.Args()...)

	if res == nil || res.OK() {
		ctx.Code = http.StatusOK
	} else {
		ctx.Code = http.StatusInternalServerError
	}

	if len(urls) > 0 {
		text = fmt.Sprintf(`<script type="text/javascript">alert("%s"); location.href="%s";</script>`, text, urls[0])
	} else {
		text = fmt.Sprintf(`<script type="text/javascript">alert("%s"); history.back();</script>`, text)
	}
	ctx.Script(text)
}

//展示通用的提示页面
func (ctx *Access) Show(res Res, urls ...string) {
	code := mBasic.Code(res.Text(), res.Code())
	text := ctx.String(res.Text(), res.Args()...)

	if res == nil || res.OK() {
		ctx.Code = http.StatusOK
	} else {
		ctx.Code = http.StatusInternalServerError
	}

	m := Map{
		"code": code,
		"text": text,
		"url":  "",
	}
	if len(urls) > 0 {
		m["url"] = urls[0]
	}

	ctx.Data["show"] = m
	ctx.View("show")
}

//返回操作结果，表示成功
//比如，登录，修改密码，等操作类的接口， 成功的时候，使用这个，
//args表示返回给客户端的data
//data 强制改为json格式，因为data有统一加密的可能
//所有数组都要加密。
func (ctx *Access) Answer(res Res, args ...Map) {
	//如果已经存在了httpDownBody，那还要把原有的reader关闭
	//释放资源， 当然在file.base.close中也应该关闭已经打开的资源
	if vv, ok := ctx.Body.(httpBufferBody); ok {
		vv.buffer.Close()
	}

	code := 0
	text := ""
	if res != nil {
		code = mBasic.Code(res.Text(), res.Code())
		text = ctx.String(res.Text(), res.Args()...)
	}

	if res == nil || res.OK() {
		ctx.Code = http.StatusOK
	} else {
		ctx.Code = http.StatusInternalServerError
	}

	// var data Map
	// if len(args) > 0 && args[0] != nil {
	// 	if data == nil {
	// 		data = make(Map)
	// 	}
	// 	for k, v := range args[0] {
	// 		data[k] = v
	// 	}
	// } else {
	// 	has := false
	// 	for range ctx.Data {
	// 		has = true
	// 		break
	// 	}
	// 	if has {
	// 		data = make(Map)
	// 		for k, v := range ctx.Data {
	// 			data[k] = v
	// 		}
	// 	}
	// }

	//20211203更新，先使用data，再使用args覆盖
	var data Map
	hasData := false
	for range ctx.Data {
		hasData = true
		break
	}
	if hasData {
		data = make(Map)
		for k, v := range ctx.Data {
			data[k] = v
		}
	}
	if len(args) > 0 && args[0] != nil {
		if data == nil {
			data = make(Map)
		}
		for k, v := range args[0] {
			data[k] = v
		}
	}

	//回写进ctx.Data
	for k, v := range data {
		ctx.Data[k] = v
	}

	ctx.Type = "json"
	ctx.Body = httpApiBody{code, text, data}
}

//通用方法
func (ctx *Access) UserAgent() string {
	return ctx.Header("User-Agent")
}
func (ctx *Access) Ip() string {
	ip := "127.0.0.1"

	if forwarded := ctx.request.Header.Get("x-forwarded-for"); forwarded != "" {
		ip = forwarded
	} else if realIp := ctx.request.Header.Get("X-Real-IP"); realIp != "" {
		ip = realIp
	} else {
		ip = ctx.request.RemoteAddr
	}

	newip, _, err := net.SplitHostPort(ip)
	if err == nil {
		ip = newip
	}

	//处理ip，可能有多个
	ips := strings.Split(ip, ", ")
	if len(ips) > 0 {
		return ips[len(ips)-1]
	}
	return ip
}

//----------------------- Token begin ---------------------------------
//手动发行token， 才会写入cookies或是从接口的token节返回
func (ctx *Access) Issue(token string) {
	ctx.issue = true
	ctx.token = token
	verify, err := mToken.Validate(token)
	if err == nil {
		ctx.verify = verify
	}
}

func (ctx *Access) Auth(authorized bool, identity string, payload Map, expires ...time.Duration) string {
	data := &Token{
		Authorized: authorized, Identity: identity, Payload: payload,
	}
	if tid := ctx.ActId(); tid != "" {
		data.ActId = tid
	}

	token, err := mToken.Sign(data, expires...)
	if err != nil {
		ctx.lastError = errorResult(err)
		return ""
	}

	//不自动下发了
	// if err == nil {
	// 	//标识为颁发新token
	// 	ctx.issue = true
	// 	ctx.token = token
	// 	ctx.verify = data
	// }

	return token
}
func (ctx *Access) AuthIssue(authorized bool, identity string, payload Map, expires ...time.Duration) string {
	token := ctx.Auth(authorized, identity, payload, expires...)
	ctx.Issue(token)
	return token
}
