package main

import (
	"github.com/mattn/go-session-manager"
	"log"
	"os"
	"strings"
	"template"
	"web"
)

const page = `
<!doctype html>
<html>
<meta charset="utf-8"/>
<body>
{.section session}
{.section Value}
Hi {@}.
<form method="post" action="/logout">
<input type="submit" name="method" value="logout" />
</form>
You will logout after 10 seconds. Then try to reload.
{.or}
<form method="post" action="/login">
<label for="name">Name:</label>
<input type="text" id="name" name="name" value="" />
<input type="submit" name="method" value="login" />
</form>
{.end}
{.end}
</body>
</html>
`

var fmap = template.FormatterMap{"html": template.HTMLFormatter}
var tmpl = template.MustParse(page, fmap)

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	manager := session.NewSessionManager(logger)
	manager.OnStart(func(session *session.Session) {
		println("started new session")
	})
	manager.OnEnd(func(session *session.Session) {
		println("abandon")
	})
	manager.SetTimeout(10)

	GetSession := func(ctx *web.Context) *session.Session {
		id, _ := ctx.GetSecureCookie("SessionId")
		session := manager.GetSessionById(id)
		ctx.SetSecureCookie("SessionId", session.Id, int64(manager.GetTimeout()))
		ctx.SetHeader("Pragma", "no-cache", true)
		return session
	}

	web.Config.CookieSecret = "7C19QRmwf3mHZ9CPAaPQ0hsWeufKd"
	web.Get("/", func(ctx *web.Context) {
		session := GetSession(ctx)
		tmpl.Execute(ctx, map[string]interface{}{"session": session})
	})
	web.Post("/login", func(ctx *web.Context) {
		name := strings.Trim(ctx.Params["name"], " ")
		if name != "" {
			logger.Printf("User \"%s\" login", name)

			// XXX: set user own object.
			GetSession(ctx).Value = name
		}
		ctx.Redirect(302, "/")
	})
	web.Post("/logout", func(ctx *web.Context) {
		session := GetSession(ctx)
		if session.Value != nil {
			// XXX: get user own object.
			logger.Printf("User \"%s\" logout", session.Value.(string))
			session.Abandon()
		}
		ctx.Redirect(302, "/")
	})
	web.Run(":6061")
}
