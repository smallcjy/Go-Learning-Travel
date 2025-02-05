package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	MapWork = iota
	ReduceWork
	Wait
	End
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}


//
// main/mrworker.go calls this function.
//

// 开两个协程模拟多主机运行worker任务
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

	// call for task 
	var wg sync.WaitGroup;
	for {
		var taskinfo *TaskInfo;
		callfortaskreply := CallForTaskReply{};
		callfortaskreply.hastask = false;
		callfortaskargs := CallForTaskArgs{};
		ok := call("Coordinator.CallForTask", &callfortaskargs, &callfortaskreply);
		if ok {
			taskinfo = &callfortaskreply.taskinfo;
		} else {
			fmt.Printf("Coordinator.CallForTask failed!\n");
		}

		if !callfortaskreply.hastask {
			continue;
		}

		role := taskinfo.role;
		switch role {
		case MapWork :
			wg.Add(1);
			go DoMap(taskinfo, mapf, &wg);
			break;
		case ReduceWork :
			wg.Add(1);
			go DoReduce(taskinfo, reducef, &wg);
			break;
		case Wait :
			time.Sleep(time.Duration(time.Second * 3))
			break;
		case End :
			log.Println("No more task. Over")
			// wait all worker finish job
			wg.Wait();
			return;
		}
	}

}

// Map执行函数
func DoMap(taskinfo *TaskInfo, mapf func(string, string) []KeyValue, wg *sync.WaitGroup) {
	// read content from file
	file, err := os.Open(taskinfo.filename);
	if err != nil {
		log.Fatalf("cannot open %v", taskinfo.filename);
	}
	content, err := ioutil.ReadAll(file);
	if err != nil {
		log.Fatalf("cannot read %v", taskinfo.filename);
	}
	file.Close();

	// call map function
	kv_array := mapf(taskinfo.filename, string(content));

	// create intermediate files 相同的单词会存放在相同的中间文件中
	intermediate := make([][]KeyValue, taskinfo.R);
	for _, kv := range kv_array {
		r := ihash(kv.Key) % taskinfo.R;
		intermediate[r] = append(intermediate[r], kv);
		taskinfo.r = r; // 保存r的值
	}

	for r := 0; r < taskinfo.R; r++ {
		oname := fmt.Sprintf("mr-%v-%v", taskinfo.m, r);
		// 生成文件的原子性 / atomicWriteFile
		ofile, _ := os.CreateTemp("./", oname + "-temp");
		
		enc := json.NewEncoder(ofile);
		for _, kv := range intermediate[r] {
			err := enc.Encode(&kv);
			if err != nil {
				log.Fatalf("cannot encode %v", kv);
			}
		}

		os.Rename(ofile.Name(), oname);
		ofile.Close();
	}

	args := CallTaskDoneArgs{*taskinfo};
	reply := CallTaskDoneReply{};
	ok := call("Coordinator.CallTaskDone", &args, &reply);
	if !ok {
		log.Panicln("Coordinator.CallTaskDone Failed!")
	}

	wg.Done();
}

// Reduce执行函数
func DoReduce(taskinfo *TaskInfo, reducef func(string, []string) string, wg *sync.WaitGroup) {
	r := taskinfo.r;
	// open intermediate file mr.m.r
	iname := fmt.Sprintf("mr-/*-%v", r);
	ifile, err := os.Open(iname);
	if err != nil {
		log.Fatalf("reduce task cannot open %v", iname);
	}

	// read intermediate file
	decoder := json.NewDecoder(ifile);
	k_map := make(map[string][]string);
	for decoder.More() {
		var kv KeyValue;
		err := decoder.Decode(&kv);
		if err != nil {
			log.Fatalf("reduce task cannot decode %v", kv);
		}
		k_map[kv.Key] = append(k_map[kv.Key], kv.Value);
	}

	ifile.Close();
	
	// 对所有的key进行DoReduce, 并写入到文件中
	oname := fmt.Sprintf("mr-out-%v", r);
	ofile, _ := os.CreateTemp("./", oname + "-temp");

	for key, values := range k_map {
		key_time := fmt.Sprintf("%v %v\n", key, reducef(key, values));  
		ofile.WriteString(key_time);
	}

	os.Rename(ofile.Name(), oname);
	ofile.Close();

	wg.Done();
}

func Map(filename string, contents string) []KeyValue {
	ff := func(r rune) bool { return !unicode.IsLetter(r)};

	// splite content into array of words
	words := strings.FieldsFunc(contents, ff);

	kv_array := []KeyValue{};
	for _, w := range words {
		kv := KeyValue{w, "1"};
		kv_array = append(kv_array, kv);
	}

	return kv_array;
}

/// reduce 函数的作用是：map task会产生key， 每个key对应一个value list， reduce函数会对这个value list进行处理，即统计values的长度
func Reduce(key, values []string) string {
	return strconv.Itoa(len(values));
}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	// sockname := coordinatorSock()
	// c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}