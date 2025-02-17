package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/gaowei-space/markdown-blog/internal/utils"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/urfave/cli"
)

var (
	MdDir       string
	Env         string
	Title       string
	Index       string
	LayoutFile  = "web/views/layouts/layout.html"
	ArticlesDir = "cache/articles/"
	LogsDir     = "cache/logs/"
	AssetsDir   = "web/assets"
)

// web服务器默认端口
const DefaultPort = 5006

func initParams(ctx *cli.Context) {
	MdDir = ctx.String("dir")
	Env = ctx.String("env")
	Title = ctx.String("title")
	Index = ctx.String("index")
}

func setLog(app *iris.Application) {
	os.MkdirAll(LogsDir, 0777)
	f, _ := os.OpenFile(LogsDir+"access-"+time.Now().Format("20060102")+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)

	if Env == "prod" {
		app.Logger().SetOutput(f)
	} else {
		app.Logger().SetLevel("debug")
		app.Logger().Debugf(`Log level set to "debug"`)
	}

	// Close the file on shutdown.
	app.ConfigureHost(func(su *iris.Supervisor) {
		su.RegisterOnShutdown(func() {
			f.Close()
		})
	})

	ac := accesslog.New(f)
	ac.AddOutput(app.Logger().Printer)
	app.UseRouter(ac.Handler)
	app.Logger().Debugf("Using <%s> to log requests", f.Name())
}

func RunWeb(ctx *cli.Context) {
	initParams(ctx)
	app := iris.New()

	setLog(app)

	tmpl := iris.HTML("./", ".html").Reload(true)
	app.RegisterView(tmpl)
	app.OnErrorCode(iris.StatusNotFound, notFound)
	app.OnErrorCode(iris.StatusInternalServerError, internalServerError)

	setIndexAuto := false
	if Index == "" {
		setIndexAuto = true
	}

	app.Use(func(ctx iris.Context) {
		navs, firstNav := getNavs()

		firstLink := strings.TrimPrefix(firstNav.Link, "/")
		if setIndexAuto && Index != firstLink {
			Index = firstLink
		}

		ctx.ViewData("Title", Title)
		ctx.ViewData("Nav", navs)
		ctx.ViewData("ActiveNav", getActiveNav(ctx))
		ctx.ViewLayout(LayoutFile)

		ctx.Next()
	})

	tmpl.AddFunc("inc", utils.Inc)

	tmpl.AddFunc("getActive", utils.GetActive)

	app.HandleDir("/static", AssetsDir)

	// TODO 增加环境变量，获取缓存时间
	app.Get("/{f:path}", iris.Cache(time.Minute*0), show)

	app.Run(iris.Addr(":" + strconv.Itoa(parsePort(ctx))))
}

func parsePort(ctx *cli.Context) int {
	port := DefaultPort
	if ctx.IsSet("port") {
		port = ctx.Int("port")
	}
	if port <= 0 || port >= 65535 {
		port = DefaultPort
	}

	return port
}

func getNavs() ([]map[string]interface{}, utils.Node) {
	var option utils.Option
	option.RootPath = []string{MdDir}
	option.SubFlag = true
	option.IgnorePath = []string{`.git`}
	option.IgnoreFile = []string{`.DS_Store`, `.gitignore`, `README.md`}
	tree, _ := utils.Explorer(option)

	navs := make([]map[string]interface{}, 0)
	for _, v := range tree.Children {
		for _, item := range v.Children {
			navs = append(navs, structs.Map(item))
		}
	}

	firstNav := getFirstNav(*tree.Children[0])

	return navs, firstNav
}

func getFirstNav(node utils.Node) utils.Node {
	if !node.IsDir {
		return node
	}
	return getFirstNav(*node.Children[0])
}

func getActiveNav(ctx iris.Context) string {
	f := ctx.Params().Get("f")
	if f == "" {
		f = Index
	}
	return f
}

func show(ctx iris.Context) {
	f := getActiveNav(ctx)
	mdfile := MdDir + f + ".md"
	articlefile := ArticlesDir + f + ".html"

	_, err := os.Stat(mdfile)
	if err != nil {
		ctx.StatusCode(404)
		ctx.Application().Logger().Errorf("Not Found '%s', Path is %s", mdfile, ctx.Path())
		return
	}

	content, err := os.ReadFile(mdfile)
	if err != nil {
		ctx.StatusCode(500)
		ctx.Application().Logger().Errorf("ReadFile Error '%s', Path is %s", mdfile, ctx.Path())
		return
	}

	os.MkdirAll(filepath.Dir(articlefile), 0777)

	unsafe := blackfriday.MarkdownCommon(content)
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	if err := os.WriteFile(articlefile, html, 0777); err != nil {
		ctx.StatusCode(500)
		ctx.Application().Logger().Errorf("WriteFile Error %s, Path is %s", err, ctx.Path())
		return
	}

	ctx.View(articlefile)
}

func notFound(ctx iris.Context) {
	ctx.View("web/views/errors/404.html")
}

func internalServerError(ctx iris.Context) {
	ctx.View("web/views/errors/500.html")
}
