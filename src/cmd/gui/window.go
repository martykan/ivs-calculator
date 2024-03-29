/*
 * Copyright (c) 2021 Tomas Martykan
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, version 3.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
 * General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"fmt"
	"ivs-calculator/pkg/interpreter"
	"log"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

/**
 * Keeps track of window state data
 */
type WindowState struct {
	sheet            *gtk.Grid
	scrollWindow     *gtk.ScrolledWindow
	oldTextInputs    []*gtk.TextView
	textInput        *gtk.TextView
	shouldScrollDown int
	buttonPressTime  time.Time
}

/**
 * Create the sheet with calculation history
 */
func (state *WindowState) createSheet() {
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetHExpand(true)
	sw.SetVExpand(true)
	grid, _ := gtk.GridNew()
	grid.SetHExpand(true)
	grid.SetVExpand(true)
	sw.Add(grid)
	grid.Connect("size-allocate", func() {
		// REMARK: This unfortunately needs to be called several times to work correctly
		if state.shouldScrollDown <= 0 {
			return
		}
		state.shouldScrollDown--
		adjustment := state.scrollWindow.GetVAdjustment()
		adjustment.SetValue(adjustment.GetUpper())
		state.scrollWindow.SetVAdjustment(adjustment)
		state.textInput.GrabFocus()
	})
	state.sheet = grid
	state.scrollWindow = sw
}

/**
 * Create a new text input in the history sheet
 */
func (state *WindowState) createTextInput() {
	textView, _ := gtk.TextViewNew()
	styleContext, _ := textView.GetStyleContext()
	styleContext.AddClass("calculator-textinput")
	textView.SetHExpand(true)
	textView.Connect("key-press-event", state.inputCallback)
	if state.textInput != nil {
		state.oldTextInputs = append(state.oldTextInputs, state.textInput)
	}
	state.textInput = textView
	state.sheet.Attach(textView, 0, len(state.oldTextInputs), 1, 1)
}

/**
 * Create a calculator key button
 * @param label Label of the button
 */
func (state *WindowState) createButton(label string) *gtk.Button {
	button, _ := gtk.ButtonNew()
	button.SetLabel(label)
	styleContext, _ := button.GetStyleContext()
	styleContext.AddClass("calculator-button")
	if label == "=" {
		styleContext.AddClass("calculator-equals")
	} else if label == "?" {
		styleContext.AddClass("calculator-help")
		image, err := gtk.ImageNewFromIconName("dialog-question-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
		if err == nil {
			button.SetLabel("")
			button.SetImage(image)
			button.SetAlwaysShowImage(true)
		}
	} else if strings.Index("0123456789", label) > -1 {
		styleContext.AddClass("calculator-num")
	} else if strings.Index("+-*/", label) > -1 {
		styleContext.AddClass("calculator-op")
	}
	if label == "" {
		button.SetSensitive(false)
	}
	button.Connect("button-press-event", func() {
		state.buttonPressTime = time.Now()
	})
	button.Connect("clicked", func() {
		state.buttonCallback(label)
	})
	return button
}

/**
 * Callback from button click event
 * @param label Label of the button
 */
func (state *WindowState) buttonCallback(label string) {
	log.Printf("Button %s", label)
	if label == "=" {
		state.finishCalculation()
		return
	}
	buffer, _ := state.textInput.GetBuffer()
	switch label {
	case "CE/C":
		if buffer.GetHasSelection() {
			buffer.DeleteSelection(false, false)
		} else if time.Since(state.buttonPressTime).Milliseconds() > 300 {
			buffer.SetText("")
		} else {
			cursorPosProp, _ := buffer.GetProperty("cursor-position")
			cursorPos := cursorPosProp.(int)
			if cursorPos == 0 {
				return
			}
			startIter := buffer.GetStartIter()
			startIter.ForwardCursorPositions(cursorPos - 1)
			endIter := buffer.GetStartIter()
			endIter.ForwardCursorPositions(cursorPos)
			buffer.Delete(startIter, endIter)
		}
	case "|  |":
		buffer.InsertAtCursor("|")
	case "?":
		glib.IdleAdd(func() {
			showHelp()
		})
	default:
		buffer.InsertAtCursor(label)
	}
	state.textInput.GrabFocus()
}

/**
 * Callback from text input keypress event
 */
func (state *WindowState) inputCallback(textView *gtk.TextView, ev *gdk.Event) bool {
	keyEvent := &gdk.EventKey{ev}
	if keyEvent.KeyVal() == gdk.KEY_Return {
		state.finishCalculation()
		return true
	}
	return false
}

/**
 * Perform calculation and handle the result
 */
func (state *WindowState) finishCalculation() {
	input := TextView_GetText(state.textInput)
	if input == "" {
		return
	}
	// Async
	go func() {
		input = strings.ReplaceAll(input, "r", "√")
		input = strings.ReplaceAll(input, "p", "^")
		node, err := interpreter.Parse(input)
		if err != nil {
			state.showCalculationError(fmt.Sprintf("syntax error at position %d", err[0]))
			return
		}
		floatResult, err2 := interpreter.Interpret(node)
		if err2 != nil {
			state.showCalculationError(err2.Error())
			return
		}
		result := fmt.Sprintf("%g", floatResult)
		state.showCalculationResult(result)
	}()
}

/**
 * Show calculation result
 */
func (state *WindowState) showCalculationResult(result string) {
	glib.IdleAdd(func() {
		state.textInput.SetEditable(false)
		styleContext, _ := state.textInput.GetStyleContext()
		styleContext.AddClass("calculator-textinput-finished")
		state.createTextInput()
		TextView_SetText(state.textInput, result)
		state.textInput.SetEditable(false)
		state.textInput.SetJustification(gtk.JUSTIFY_RIGHT)
		styleContext, _ = state.textInput.GetStyleContext()
		styleContext.AddClass("calculator-textinput-result")
		state.createTextInput()
		state.scrollWindow.ShowAll()
		state.shouldScrollDown = 3
	})
}

/**
 * Show calculation error message
 */
func (state *WindowState) showCalculationError(err string) {
	glib.IdleAdd(func() {
		state.textInput.SetEditable(false)
		styleContext, _ := state.textInput.GetStyleContext()
		styleContext.AddClass("calculator-textinput-finished")
		state.createTextInput()
		TextView_SetText(state.textInput, err)
		state.textInput.SetEditable(false)
		state.textInput.SetJustification(gtk.JUSTIFY_RIGHT)
		styleContext, _ = state.textInput.GetStyleContext()
		styleContext.AddClass("calculator-textinput-error")
		state.createTextInput()
		state.scrollWindow.ShowAll()
		state.shouldScrollDown = 3
	})
}

/**
 * Create the app layout and initialize WindowState
 */
func createLayout() *gtk.Grid {
	state := WindowState{}
	state.createSheet()
	state.createTextInput()

	grid, _ := gtk.GridNew()
	grid.Attach(state.scrollWindow, 0, 0, 5, 1)

	buttonLabels := [5][5]string{
		{"√", "(", ")", "CE/C", "/"},
		{"^", "7", "8", "9", "*"},
		{"!", "4", "5", "6", "-"},
		{"%", "1", "2", "3", "+"},
		{"|  |", "0", ",", "?", "="},
	}
	for i := 0; i < 25; i++ {
		label := buttonLabels[i/5][i%5]
		grid.Attach(state.createButton(label), i%5, 1+i/5, 1, 1)
	}

	grid.SetHExpand(true)
	grid.SetVExpand(true)
	return grid
}

/**
 * Create the Gtk Window
 */
func createWindow() {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}
	win.SetTitle("IVS Calculator")
	win.SetDefaultSize(800, 600)
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	// Styling
	cssProvider, _ := gtk.CssProviderNew()
	data, _ := Asset("res/style.css")
	cssProvider.LoadFromData(string(data))
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	styleContext, _ := win.GetStyleContext()
	styleContext.AddProvider(cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)

	// Create layout
	win.Add(createLayout())
	win.ShowAll()
}

/**
 * Main function
 */
func main() {
	gtk.Init(nil)
	createWindow()
	gtk.Main()
}
