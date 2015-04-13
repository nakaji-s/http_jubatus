package process

import (
	"bufio"
	"fmt"
	"github.com/ugorji/go/codec"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

type JubatusProcess struct {
	cmd    *exec.Cmd
	Port   int
	client *rpc.Client
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

	for {
		if j.client != nil {
			args := replaceFloat64To32(append(name, arg...))
			err := j.client.Call(method, codec.MsgpackSpecRpcMultiArgs(args), &result)
			if err == nil {
				result = replaceByteSliceToString([]interface{}{result})[0]

				break
			}
		}

		newClient, err := connect(fmt.Sprintf("localhost:%d", j.Port))
		if err != nil {
			return nil, err
		}
		fmt.Println("reconnected to jubatus process")
		j.client = newClient
	}
	fmt.Println("result ", result)
	return result, nil
}

func connect(target string) (*rpc.Client, error) {
	// create client
	conn, err := net.Dial("tcp", target)
	if err != nil {
		return nil, err
	}
	mh := new(codec.MsgpackHandle)
	mh.StructToArray = true
	rpcCodec := codec.MsgpackSpecRpc.ClientCodec(conn, mh)
	return rpc.NewClientWithCodec(rpcCodec), nil
}

func NewJubatusProcess(command string, filepath string) (*JubatusProcess, error) {
	/*
	   boot jubauts procsss.
	   it searches available port for jubatus from 9200
	*/
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

		client, err := connect(fmt.Sprintf("localhost:%d", port))
		if err != nil {
			return nil, err
		}
		return &JubatusProcess{cmd, port, client}, nil
	}
}

func replaceFloat64To32(src []interface{}) []interface{} {
	out := []interface{}{}
	for _, v := range src {
		if reflect.TypeOf(v).Kind() == reflect.Float64 {
			tmp_float64, _ := v.(float64)
			out = append(out, float32(tmp_float64))
		} else if reflect.TypeOf(v).Kind() == reflect.Slice && reflect.TypeOf(v).Elem().Kind() == reflect.Interface {
			tmp_slice, _ := v.([]interface{})
			out = append(out, replaceFloat64To32(tmp_slice))
		} else {
			out = append(out, v)
		}
	}
	return out
}

func replaceByteSliceToString(src []interface{}) []interface{} {
	out := []interface{}{}
	for _, v := range src {
		if reflect.TypeOf(v).Kind() == reflect.Slice {
			switch reflect.TypeOf(v).Elem().Kind() {
			case reflect.Uint8:
				tmp_str, _ := v.([]byte)
				out = append(out, string(tmp_str))
			case reflect.Interface:
				tmp_slice, _ := v.([]interface{})
				out = append(out, replaceByteSliceToString(tmp_slice))
			default:
				out = append(out, v)
			}
		} else {
			out = append(out, v)
		}
	}
	return out
}
