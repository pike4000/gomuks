// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package messages

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"maunium.net/go/gomuks/matrix/event"
	"maunium.net/go/mautrix"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/ansimage"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

type FileMessage struct {
	Type      mautrix.MessageType
	Body      string
	URL       mautrix.ContentURI
	Thumbnail mautrix.ContentURI
	imageData []byte
	buffer    []tstring.TString

	matrix ifc.MatrixContainer
}

// NewFileMessage creates a new FileMessage object with the provided values and the default state.
func NewFileMessage(matrix ifc.MatrixContainer, evt *event.Event, displayname string) *UIMessage {
	url, _ := mautrix.ParseContentURI(evt.Content.URL)
	thumbnail, _ := mautrix.ParseContentURI(evt.Content.GetInfo().ThumbnailURL)
	return newUIMessage(evt, displayname, &FileMessage{
		Type:      evt.Content.MsgType,
		Body:      evt.Content.Body,
		URL:       url,
		Thumbnail: thumbnail,
		matrix:    matrix,
	})
}

func (msg *FileMessage) Clone() MessageRenderer {
	data := make([]byte, len(msg.imageData))
	copy(data, msg.imageData)
	return &FileMessage{
		Body:      msg.Body,
		URL:       msg.URL,
		Thumbnail: msg.Thumbnail,
		imageData: data,
		matrix:    msg.matrix,
	}
}

func (msg *FileMessage) NotificationContent() string {
	switch msg.Type {
	case mautrix.MsgImage:
		return "Sent an image"
	case mautrix.MsgAudio:
		return "Sent an audio file"
	case mautrix.MsgVideo:
		return "Sent a video"
	case mautrix.MsgFile:
		fallthrough
	default:
		return "Sent a file"
	}
}

func (msg *FileMessage) PlainText() string {
	return fmt.Sprintf("%s: %s", msg.Body, msg.matrix.GetDownloadURL(msg.URL))
}

func (msg *FileMessage) String() string {
	return fmt.Sprintf(`&messages.FileMessage{Body="%s", URL="%s", Thumbnail="%s"}`, msg.Body, msg.URL, msg.Thumbnail)
}

func (msg *FileMessage) DownloadPreview() {
	url := msg.Thumbnail
	if url.IsEmpty() {
		if msg.Type == mautrix.MsgImage && !msg.URL.IsEmpty() {
			msg.Thumbnail = msg.URL
			url = msg.Thumbnail
		} else {
			return
		}
	}
	debug.Print("Loading file:", url)
	data, err := msg.matrix.Download(url)
	if err != nil {
		debug.Printf("Failed to download file %s: %v", url, err)
		return
	}
	debug.Print("File", url, "loaded.")
	msg.imageData = data
}

func (msg *FileMessage) ThumbnailPath() string {
	return msg.matrix.GetCachePath(msg.Thumbnail)
}

func (msg *FileMessage) CalculateBuffer(prefs config.UserPreferences, width int, uiMsg *UIMessage) {
	if width < 2 {
		return
	}

	if prefs.BareMessageView || prefs.DisableImages || len(msg.imageData) == 0 {
		msg.buffer = calculateBufferWithText(prefs, tstring.NewTString(msg.PlainText()), width, uiMsg)
		return
	}

	img, _, err := image.DecodeConfig(bytes.NewReader(msg.imageData))
	if err != nil {
		debug.Print("File could not be decoded:", err)
	}
	imgWidth := img.Width
	if img.Width > width {
		imgWidth = width / 3
	}

	ansFile, err := ansimage.NewScaledFromReader(bytes.NewReader(msg.imageData), 0, imgWidth, color.Black)
	if err != nil {
		msg.buffer = []tstring.TString{tstring.NewColorTString("Failed to display image", tcell.ColorRed)}
		debug.Print("Failed to display image:", err)
		return
	}

	msg.buffer = ansFile.Render()
}

func (msg *FileMessage) Height() int {
	return len(msg.buffer)
}

func (msg *FileMessage) Draw(screen mauview.Screen) {
	for y, line := range msg.buffer {
		line.Draw(screen, 0, y)
	}
}
