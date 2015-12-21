package process

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	anomaly "github.com/jubatus/jubatus-go-client/lib/anomaly"
	bandit "github.com/jubatus/jubatus-go-client/lib/bandit"
	burst "github.com/jubatus/jubatus-go-client/lib/burst"
	classifier "github.com/jubatus/jubatus-go-client/lib/classifier"
	clustering "github.com/jubatus/jubatus-go-client/lib/clustering"
	graph "github.com/jubatus/jubatus-go-client/lib/graph"
	nearest_neighbor "github.com/jubatus/jubatus-go-client/lib/nearest_neighbor"
	recommender "github.com/jubatus/jubatus-go-client/lib/recommender"
	regression "github.com/jubatus/jubatus-go-client/lib/regression"
	stat "github.com/jubatus/jubatus-go-client/lib/stat"
	"github.com/serenize/snaker"
	"github.com/ugorji/go/codec"
	"math"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

func adjust_type(arg interface{}, input_type reflect.Type) (interface{}, error) {
	data, err := json.Marshal([]interface{}{arg})
	//data, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}

	val := reflect.New(input_type).Elem().Interface()

	var ret interface{}
	// int32 is decoded to float64 if it doesn't treated
	switch t := val.(type) {
	case int32:
		st := []int32{t}
		err = json.Unmarshal(data, &st)
		ret = st
	case float64:
		st := []float64{t}
		err = json.Unmarshal(data, &st)
		ret = st
	case string:
		st := []string{t}
		err = json.Unmarshal(data, &st)
		ret = st[0]
	case []graph.ShortestPathQuery:
		st := [][]graph.ShortestPathQuery{t}
		err = json.Unmarshal(data, &st)
		ret = st
	default:
		st := []interface{}{}
		err = json.Unmarshal(data, &st)
		ret = st[0]
	}

	if err != nil {
		fmt.Println("error:", err)
		return nil, err
	}

	return ret, nil
}

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

func (j *JubatusProcess) Call(module_name string, method_name string, args []interface{}) (interface{}, error) {
	modules := map[string]interface{}{
		"anomaly":          &anomaly.AnomalyClient{},
		"bandit":           &bandit.BanditClient{},
		"burst":            &burst.BurstClient{},
		"classifier":       &classifier.ClassifierClient{},
		"clustering":       &clustering.ClusteringClient{},
		"graph":            &graph.GraphClient{},
		"nearest_neighbor": &nearest_neighbor.NearestNeighborClient{},
		"recommender":      &recommender.RecommenderClient{},
		"regression":       &regression.RegressionClient{},
		"stat":             &stat.StatClient{},
	}

	fmt.Println("calling ", method_name, " ", args, " port:", j.Port)

	in_args := []interface{}{"name"}

	// find method from go-client
	camel_method_name := snaker.SnakeToCamel(method_name)
	method := reflect.ValueOf(modules[module_name]).MethodByName(camel_method_name)
	if method.Kind() == reflect.Invalid {
		return nil, errors.New("method not found")
	}

	// create args(adjust type)
	for i, arg := range args {
		ad_arg, err := adjust_type(arg, method.Type().In(i))
		if err != nil {
			return nil, err
		}
		in_args = append(in_args, ad_arg)
	}

	var result interface{}

	for {
		if j.client != nil {
			val := reflect.New(method.Type().Out(0)).Elem().Interface()
			switch val.(type) {
			case int32:
				var ret int32
				j.client.Call(method_name, codec.MsgpackSpecRpcMultiArgs(in_args), &ret)
				result = ret
			case float64:
				var ret float64
				j.client.Call(method_name, codec.MsgpackSpecRpcMultiArgs(in_args), &ret)
				if math.IsInf(ret, 0) == true {
					result = fmt.Sprintf("%e", ret)
				} else {
					result = ret
				}
			case burst.Window:
				var ret burst.Window
				j.client.Call(method_name, codec.MsgpackSpecRpcMultiArgs(in_args), &ret)
				result = ret
			case map[string]burst.Window:
				var ret map[string]burst.Window
				j.client.Call(method_name, codec.MsgpackSpecRpcMultiArgs(in_args), &ret)
				result = ret
			case map[string]bandit.ArmInfo:
				var ret map[string]bandit.ArmInfo
				j.client.Call(method_name, codec.MsgpackSpecRpcMultiArgs(in_args), &ret)
				result = ret
			case [][]classifier.EstimateResult:
				var ret [][]classifier.EstimateResult
				j.client.Call(method_name, codec.MsgpackSpecRpcMultiArgs(in_args), &ret)
				result = ret
			default:
				j.client.Call(method_name, codec.MsgpackSpecRpcMultiArgs(in_args), &result)
			}

			break
		}

		newClient, err := connect(fmt.Sprintf("localhost:%d", j.Port))
		if err != nil {
			return nil, err
		}
		fmt.Println("reconnected to jubatus process")
		j.client = newClient
	}

	if result != nil {
		result = replaceByteSliceToString([]interface{}{result})[0]
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
