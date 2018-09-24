package main

import (
	"log"
	"time"

	"net"
	"strings"
	"fmt"
	"github.com/gen2brain/beeep"
	"github.com/spf13/viper"
	"github.com/marcusolsson/tui-go"
	"strconv"
)

var (
	username         string
	multicastAddress *net.UDPAddr // IPv4 Multicast Address
	bufferSize        = 1024
)

func init() {
	// config file in /home/$username/.bob/chatbob.yaml
	// or  chatbob.yaml in the current working directory
	// values that needs to be set:
	//	ip (string, multicast IP)
	//	port (int)
	//	username (string)
	viper.SetConfigName("chatbob")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.bob")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}

func encodeMessage(message string) []byte {
	return []byte(username + "|" + message)
}

func decodeMessage(data []byte, n int) (string, string) {
	fields := strings.Split(string(data[:n]), "|")
	return fields[0], fields[1]
}

func sendMessage(message string) error {
	c, err := net.DialUDP("udp", nil, multicastAddress)
	defer c.Close()
	if err != nil {
		return err
	}
	_, err = c.Write(encodeMessage(message))
	return err
}

func buildEntry(username, message string) *tui.Box {
	return tui.NewHBox(
		tui.NewLabel(time.Now().Format("15:04")),
		tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", username))),
		tui.NewLabel(message),
		tui.NewSpacer(),
	)
}

func main() {
	var err error
	username = viper.GetString("username")
	multicastAddress, err = net.ResolveUDPAddr("udp", viper.GetString("ip")+":"+strconv.Itoa(viper.GetInt("port")))
	if err != nil {
		panic(fmt.Errorf("Fatal error resolving multicast address %s \n", err))
	}

	history := tui.NewVBox()

	historyScroll := tui.NewScrollArea(history)
	historyScroll.SetAutoscrollToBottom(true)

	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {
		sendMessage(e.Text())
		input.SetText("")
	})

	root := tui.NewHBox(chat)

	ui, err := tui.New(root)
	if err != nil {
		log.Fatal(err)
	}

	ui.SetKeybinding("Esc", func() { ui.Quit() })

	go func() {
		conn, err := net.ListenMulticastUDP("udp", nil, multicastAddress)
		if err != nil {
			log.Fatal(err)
		}

		conn.SetReadBuffer(bufferSize)

		for {
			b := make([]byte, bufferSize)
			n, _, err := conn.ReadFromUDP(b)
			if err != nil {
				log.Fatal("ReadFromUDP failed:", err)
			}

			sender, message := decodeMessage(b, n)

			ui.Update(func() {
				history.Append(buildEntry(sender, message))
			})

			if sender != username {
				beeep.Notify(sender, message, "")
			}
		}
	}()

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
