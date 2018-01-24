// Copyright 2017 Philipp Resch <phil@2kd.de>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import (
	"strings"

	termbox "github.com/nsf/termbox-go"
)

var activeDialogItem int

const (
	DialogText = iota
	DialogInputField
	DialogConfirmButton
	DialogCancelButton
)

type DialogItem struct {
	Header      string
	Name        string
	Value       string
	ItemType    int
	ItemFgColor Attribute
	ItemBgColor Attribute
	inputPosX   int
	inputPosY   int
}

// Dialog is the main struct
type Dialog struct {
	Block
	DialogItems   []DialogItem
	DialogText    string
	DialogFgColor Attribute
	DialogBgColor Attribute
	Name          string
	IsCapturing   bool
}

// NewDialog returns a new *Dialog with current theme.
func NewDialog() *Dialog {
	d := &Dialog{Block: *NewBlock()}
	d.DialogFgColor = ThemeAttr("par.text.fg")
	d.DialogBgColor = ThemeAttr("par.text.bg")
	activeDialogItem = 0
	return d
}

// Buffer implements Bufferer interface.
func (d *Dialog) Buffer() Buffer {
	buf := d.Block.Buffer()
	iOffset := 0

	// Write header
	if len(d.DialogText) > 0 {
		d.writeText(&buf, d.DialogText, 1, 1, d.DialogFgColor, d.DialogBgColor)
		iOffset = 3
	}

	// Write the elements
	for i, v := range d.DialogItems {
		switch v.ItemType {
		case DialogInputField:
			d.writeText(&buf, v.Header, i+iOffset, 1, v.ItemFgColor, v.ItemBgColor)
			d.writeText(&buf, v.Value, i+iOffset, len(v.Header)+2, v.ItemFgColor, v.ItemBgColor)

			if d.DialogItems[i].inputPosX == 0 {
				d.DialogItems[i].inputPosX = len(v.Header) + 3 + d.X
			}
			if d.DialogItems[i].inputPosY == 0 {
				d.DialogItems[i].inputPosY = i + iOffset + d.Y + 1
			}
		case DialogConfirmButton:
			d.writeText(&buf, v.Header, i+iOffset, 1, v.ItemFgColor, v.ItemBgColor)
		}

	}

	if d.IsCapturing {
		d.setActiveDialogItem(activeDialogItem)
	}

	return buf
}

// StartCapture begins catching events from the /sys/kbd stream and updates the Dialog. While
// capturing events, the Dialog also publishes its own event stream under the /input/kbd path.
func (d *Dialog) StartCapture() {
	d.IsCapturing = true
	Handle("/sys/kbd", func(e Event) {
		if d.IsCapturing {
			key := e.Data.(EvtKbd).KeyStr

			switch key {
			case "<enter>":
				// Confirm entry
				if d.DialogItems[activeDialogItem].ItemType == DialogConfirmButton {
					// send a arbitrary key
					key = "<confirm>"
					break
				}
				// Otherwise go to next entry field
				d.activateNextDialogItem()
			case "C-8", "<backspace>":
				d.backspace()
			case "<down>":
				if activeDialogItem == len(d.DialogItems)-1 {
					break
				}
				d.activateNextDialogItem()
			case "<up>":
				if activeDialogItem == 0 {
					break
				}
				d.activatePreviosDialogItem()

			case "<tab>":
				d.activateNextDialogItem()
			default:
				// If it's a CTRL something we don't handle then just ignore it
				if strings.HasPrefix(key, "C-") {
					break
				}

				d.addString(key)
			}
			if d.Name == "" {
				SendCustomEvt("/input/kbd", d.getInputEvt(key))
			} else {
				SendCustomEvt("/input/"+d.Name+"/kbd", d.getInputEvt(key))
			}

			Render(d)
		}
	})
}

// StopCapture tells the Dialog to stop accepting events from the /sys/kbd stream
func (d *Dialog) StopCapture() {
	d.IsCapturing = false
}

func (d *Dialog) GetInputFieldHeaders() []string {
	var strs []string
	for _, item := range d.DialogItems {
		strs = append(strs, item.Header)
	}
	return strs
}

func (d *Dialog) getInputEvt(key string) EvtInput {
	return EvtInput{
		KeyStr: key,
	}
}

func (d *Dialog) backspace() {
	l := len(d.DialogItems[activeDialogItem].Value)
	if l > 0 {
		d.DialogItems[activeDialogItem].Value = d.DialogItems[activeDialogItem].Value[:l-1]
		d.DialogItems[activeDialogItem].inputPosX--
	}
}

func (d *Dialog) addString(key string) {
	d.DialogItems[activeDialogItem].Value += key
	d.DialogItems[activeDialogItem].inputPosX++
}

func (d *Dialog) writeText(buf *Buffer, text string, x int, y int, fgColor Attribute, bgColor Attribute) {
	var cs []Cell

	cs = DTrimTxCls(DefaultTxBuilder.Build(text, fgColor, bgColor), d.innerArea.Dx())

	j := y
	for _, vv := range cs {
		w := vv.Width()
		buf.Set(d.innerArea.Min.X+j, d.innerArea.Min.Y+x, vv)
		j += w
	}
}

func (d *Dialog) SetActiveDialogItem(item int) {
	// Revert all items to default color
	for i, _ := range d.DialogItems {
		d.DialogItems[i].ItemFgColor = d.DialogFgColor
		d.DialogItems[i].ItemBgColor = d.DialogBgColor
	}

	if d.DialogItems[item].ItemType == DialogConfirmButton || d.DialogItems[item].ItemType == DialogCancelButton {
		termbox.HideCursor()
		d.DialogItems[item].ItemFgColor = ColorBlack
		d.DialogItems[item].ItemBgColor = ColorYellow
		return
	}

	termbox.SetCursor(d.DialogItems[item].inputPosX, d.DialogItems[item].inputPosY)
}

func (d *Dialog) activateNextDialogItem() {

	if activeDialogItem < len(d.DialogItems)-1 {
		activeDialogItem++
	} else {
		activeDialogItem = 0
	}
	d.setActiveDialogItem(activeDialogItem)
}

func (d *Dialog) activatePreviosDialogItem() {
	if activeDialogItem > 0 {
		activeDialogItem--
	} else {
		activeDialogItem = len(d.DialogItems) - 1
	}
	d.setActiveDialogItem(activeDialogItem)
}
