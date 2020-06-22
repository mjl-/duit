package main

import (
	"image"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mjl-/duit"
)

//初始化环境变量
func init() {
	path, err := filepath.Abs("./font/unifont.font")
	if err != nil {
		log.Fatal(err)
	}
	os.Setenv("font", path) //设置中文字体
}

func main() {

	dui, err := duit.NewDUI("测试中文", nil)
	if err != nil {
		log.Fatal(err)
	}
	status := &duit.Label{Text: "登录状态:状态演示..."}
	username := &duit.Field{}
	password := &duit.Field{Password: true}
	login := &duit.Button{
		Text:     "登录",
		Colorset: &dui.Primary,
		Click: func() (e duit.Event) {
			status.Text = "status: logging in ..."
			dui.MarkLayout(status)

			go func() {
				time.Sleep(1 * time.Second)
				dui.Call <- func() {
					status.Text = "status: logged in"
					password.Text = ""
					dui.MarkLayout(nil) // we're lazy, update all of the UI
				}
			}()
			return
		},
	}

	dui.Top.UI = &duit.Box{
		Padding: duit.SpaceXY(6, 4), // inset from the window
		Margin:  image.Pt(6, 4),     // space between kids in this box

		Kids: duit.NewKids(
			status,
			&duit.Grid{
				Columns: 2,
				Padding: []duit.Space{
					{Right: 6, Top: 4, Bottom: 4},
					{Left: 6, Top: 4, Bottom: 4},
				},
				Valign: []duit.Valign{duit.ValignMiddle, duit.ValignMiddle},
				Kids: duit.NewKids(
					&duit.Label{Text: "用户名"},
					username,
					&duit.Label{Text: "密码"},
					password,
				),
			},
			login,
		),
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)
		case warn, ok := <-dui.Error:
			if !ok {
				return
			}
			log.Printf("duit: %s\n", warn)
		}
	}
}
