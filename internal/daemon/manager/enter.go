package manager

import (
	"context"
	"fmt"
	"io"

	"github.com/opencurve/pigeon"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
)

func enter(r *pigeon.Request, data *EnterRequest) error {
	// websocket upgrade
	conn, err := upgrader.Upgrade(r.ResponseWriter(), r.Context.Request, nil)
	if err != nil {
		r.Logger().Error("websocket upgrade failed",
			pigeon.Field("error", err))
		return err
	}
	defer conn.Close()

	hr, err := containerExec(data.ContainerId, data.Home)
	if err != nil {
		r.Logger().Error("container exec failed",
			pigeon.Field("error", err))
		return err
	}
	defer hr.Close()
	defer func() {
		hr.Conn.Write([]byte("exit\r"))
	}()
	go func() {
		wsWriterCopy(hr.Conn, conn)
	}()
	wsReaderCopy(conn, hr.Conn)
	return nil
}

func containerExec(container, home string) (hr types.HijackedResponse, err error) {
	// init
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return
	}

	cli.NegotiateAPIVersion(ctx)
	// exec create
	ir, err := cli.ContainerExecCreate(ctx, container, types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/bash", "-c", fmt.Sprintf("cd %s; /bin/bash", home)},
		Tty:          true,
	})
	if err != nil {
		return
	}

	// attach
	hr, err = cli.ContainerExecAttach(ctx, ir.ID, types.ExecStartCheck{Detach: false, Tty: true})
	if err != nil {
		return
	}
	return
}

func wsWriterCopy(reader io.Reader, writer *websocket.Conn) {
	buf := make([]byte, 8192)
	defer writer.Close()
	for {
		nr, err := reader.Read(buf)
		if nr > 0 {
			err := writer.WriteMessage(websocket.BinaryMessage, buf[0:nr])
			if err != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func wsReaderCopy(reader *websocket.Conn, writer io.Writer) {
	defer reader.Close()
	for {
		messageType, p, err := reader.ReadMessage()
		if err != nil {
			return
		}
		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			writer.Write(p)
		}
		if messageType == websocket.CloseMessage {
			return
		}
	}
}
