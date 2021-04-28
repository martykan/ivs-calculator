package main

import (
	"fmt"
	"log"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/martykan/go-webkit2-nojs/webkit2"
)

func showHelp() {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}
	win.SetTitle("Help")
	win.SetDefaultSize(600, 1000)
	win.SetPosition(gtk.WIN_POS_CENTER)

	webView := webkit2.NewWebView()
	win.Connect("destroy", func() {
		webView.Destroy()
	})

	webView.Connect("load-failed", func() {
		fmt.Println("Load failed.")
	})

	glib.IdleAdd(func() bool {
		data, _ := Asset("res/help.html")
		webView.LoadHTML(string(data), "")
		return false
	})
	webView.SetVExpand(true)
	webView.SetHExpand(true)

	win.Add(webView.ToWidget())

	win.ShowAll()
}
