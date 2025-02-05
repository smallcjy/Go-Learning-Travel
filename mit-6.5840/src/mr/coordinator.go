package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"
)

const (
	Init = iota
	MapState 
	ReduceState
	Done
)

const OUTTIME = time.Duration(10 * time.Second)

// 任务信息
type TaskInfo struct {
	M        int    // 共M个Map任务
	R        int    // 共R个Reduce任务
	m        int    // 当前第m个Map任务
	r        int    // 当前第r个Reduce任务
	filename string // 文件名
	role     int    // 任务类型
}

// 任务状态
type TaskStat struct {
	Assign   bool     //分配状态
	Timestamp int64    //时间戳
	Done     bool     //完成状态
	TaskId   int      //任务ID
	Taskinfo TaskInfo //任务信息
}

type HeartbeatMsg struct {
	reply *CallForTaskReply
	ok    chan struct{}
}

type ReportMsg struct {
	taskinfo *TaskInfo
	reply    *bool
	ok       chan struct{}
}

type Coordinator struct {
	// Your definitions here.
	OriFiles []string   // 输入文件名集合
	NMap     int        // Map任务数量
	NReduce  int        // Reduce任务数量
	maptasks    []TaskStat // Map任务集合
	reducetasks []TaskStat // Reduce任务集合
	stat     int        // Coordinator state

	heartCh  chan HeartbeatMsg
	reportCh chan ReportMsg
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.
	if c.stat == Done {
		ret = true
	}

	return ret
}

func (c *Coordinator) DoneMap() bool {
	ret := true
	for i := 0; i < len(c.maptasks); i++ {
		if !c.maptasks[i].Done {
			ret = false
			break
		}
	}
	return ret
}

func (c *Coordinator) DoneReduce() bool {
	ret := true
	for i := 0; i < len(c.reducetasks); i++ {
		if !c.reducetasks[i].Done {
			ret = false
			break
		}
	}
	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.Init(files, nReduce)
	go c.TaskScheduler()
	c.server()
	return &c
}

// Coordinator 初始化函数
func (c *Coordinator) Init(files []string, nReduce int) {
	c.OriFiles = files
	c.NMap = len(files)
	c.NReduce = nReduce
	c.maptasks = make([]TaskStat, 0)
	c.reducetasks = make([]TaskStat, 0)
	c.stat = Init
	c.heartCh = make(chan HeartbeatMsg)
	c.reportCh = make(chan ReportMsg)

	// 初始化Map任务
	for i := 0; i < c.NMap; i++ {
		newmaptask := TaskStat{
			Assign: false,
			Timestamp: 0,
			Done:   false,
			TaskId: i,
			Taskinfo: TaskInfo{
				M:        c.NMap,
				R:        c.NReduce,
				m:        i,
				r:        -1,
				filename: c.OriFiles[i],
				role:     MapWork,
			},
		}

		c.maptasks = append(c.maptasks, newmaptask)
	}

	// 初始化Reduce任务
	for i := 0; i < c.NReduce; i++ {
		newreducetask := TaskStat {
			Assign: false,
			Timestamp: 0,
			Done: false,
			TaskId: i,
			Taskinfo: TaskInfo{
				M: c.NMap,
				m: -1,
				R: c.NReduce,
				r: i,
				filename: "",
				role: ReduceWork,
			},
		}

		c.reducetasks = append(c.reducetasks, newreducetask)
	}
}

// worker请求任务
func (c *Coordinator) CallForTask(args *CallForTaskArgs, reply *CallForTaskReply) error {
	msg := HeartbeatMsg{reply: reply, ok: make(chan struct{})}
	c.heartCh <- msg
	<-msg.ok
	return nil
}

// worker报告任务完成
func (c *Coordinator) CallTaskDone(args *CallTaskDoneArgs, reply *CallTaskDoneReply) error {
	msg := ReportMsg{taskinfo: &args.taskinfo, reply: &reply.ok, ok: make(chan struct{})}
	c.reportCh <- msg
	<-msg.ok
	return nil
}

// 任务调度器
func (c *Coordinator) TaskScheduler() {
	c.stat = MapState
	for {
		select {
		case msg := <-c.heartCh:
			// 分配任务, 根据自身状态
			switch c.stat {
			case MapState:
				for i := 0; i < c.NMap; i++ {
					if !c.maptasks[i].Assign {
						msg.reply.taskinfo = c.maptasks[i].Taskinfo
						msg.reply.hastask = true
						c.maptasks[i].Assign = true
						c.maptasks[i].Timestamp = time.Now().Unix()
						break
					} else if time.Now().Unix() - c.maptasks[i].Timestamp > int64(OUTTIME) {
						// work超时
						msg.reply.taskinfo = c.maptasks[i].Taskinfo
						msg.reply.hastask = true
						c.maptasks[i].Assign = true
						c.maptasks[i].Timestamp = time.Now().Unix()
						break
					}
				}
			case ReduceState:
				for i := 0; i < c.NReduce; i++ {
					if !c.reducetasks[i].Assign {
						msg.reply.taskinfo = c.reducetasks[i].Taskinfo
						msg.reply.hastask = true
						c.reducetasks[i].Timestamp = time.Now().Unix()
					} else if time.Now().Unix() - c.reducetasks[i].Timestamp > int64(OUTTIME) {
						// work超时
						msg.reply.taskinfo = c.reducetasks[i].Taskinfo
						msg.reply.hastask = true
						c.reducetasks[i].Timestamp = time.Now().Unix()
						break
					}
				}
			}

			msg.ok <- struct{}{}
		case msg := <-c.reportCh:
			// 处理任务完成
			switch msg.taskinfo.role {
			case MapWork:
				// 标记任务完成
				c.maptasks[msg.taskinfo.m].Done = true
				if c.DoneMap() {
					c.stat = ReduceState
				}
			case ReduceWork:
				// 标记任务完成
				c.reducetasks[msg.taskinfo.r].Done = true
				if c.DoneReduce() {
					c.stat = Done
				}
			}
			msg.ok <- struct{}{}
		}
	}
}
