package tools

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"

	"github.com/opencurve/curveadm/cli/cli"
)

func EnterContainer(curveadm *cli.CurveAdm, addr, containerId, home string) (err error) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: addr, RawQuery: fmt.Sprintf("method=%s&ContainerId=%s&Home=%s", "enter", containerId, home)}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		curveadm.WriteOutln("websocket Dial, err:%s", err)
		return err
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer c.Close()
		for {
			inputReader := bufio.NewReader(curveadm.In())
			input, err := inputReader.ReadBytes('\n')
			if err != nil {
				return
			}
			if string(input) == "exit" {
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					return
				}
				return
			}
			err = c.WriteMessage(websocket.TextMessage, input)
			if err != nil {
				return
			}
		}

	}()

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			curveadm.WriteOut("%s", message)
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case sig := <-interrupt:
			if sig == syscall.SIGINT {
				err = c.WriteMessage(websocket.TextMessage, []byte{'\003'})
				if err != nil {
					return err
				}
			}
		}
	}
}
