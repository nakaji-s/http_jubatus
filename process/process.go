package process

import (
	"net"
	"net/rpc"
	"fmt"
	"os/exec"
	"strings"
	"bufio"
	"os"
	"github.com/ugorji/go/codec"
)

type JubatusProcess struct {
	cmd  *exec.Cmd
	Port int
	client rpc.Client
}

func jubatus_keep(target exec.Cmd) {
	for {
		target.Wait()
		target.Start()
	}
}

type JubatusProcessError struct {
	message string
}

func (j JubatusProcessError) Error() string {
	return fmt.Sprintf("Jubatus Process Error: %s", j.message)
}

func (j *JubatusProcess) Kill() error {
	if err := j.cmd.Process.Kill(); err != nil {
		return err
	}
	return nil
}

func (j *JubatusProcess) Call(method string, arg []interface{}) (interface{}, error) {
	var result interface{}
	fmt.Println("calling ", method, " ", arg, " port:", j.Port)
	name := []interface{}{"name"}
	j.client.Call(method, codec.MsgpackSpecRpcMultiArgs(append(name, arg...)), &result)
	fmt.Println("result ", result)
	return result, nil
}

func NewJubatusProcess(command string, filepath string) (*JubatusProcess, error) {
	port := 9200
	for {
		cmd := exec.Command(command, "-f", filepath, "-p", fmt.Sprintf("%d", port))
		cmd.Stderr = os.Stderr

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println("failed to create pipe")
			return nil, err
		}

		if err := cmd.Start(); err != nil {
			fmt.Println("failed to boot jubatus")
			return nil, err
		}

		r := bufio.NewReader(stdout)
		finish := false
		for {
			byteline, _, err := r.ReadLine()
			if err != nil {
				fmt.Println("failed to read line ", err)
				return nil, err
			}
			line := string(byteline)
			fmt.Println(line)
			if strings.Contains(line, "RPC server startup") {
				finish = true
				break
			} else if strings.Contains(line, "server failed to start") {
				port += 1
				break
			} else if strings.Contains(line, "FATAL") {
				fmt.Println("fatal jubatus error")
				return nil, JubatusProcessError{line}
			}
		}

		if !finish {
			continue
		}

		// create client
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			return nil, err
		}
		mh := new(codec.MsgpackHandle)
		mh.StructToArray = true
		rpcCodec := codec.MsgpackSpecRpc.ClientCodec(conn, mh)
		client := rpc.NewClientWithCodec(rpcCodec)

		return &JubatusProcess{cmd, port, *client}, nil
	}
}
