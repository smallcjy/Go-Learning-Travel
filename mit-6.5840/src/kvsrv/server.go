package kvsrv

import (
	"log"
	"sync"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type Result struct {
	requestId int64
	value string
}

type KVServer struct {
	mu sync.Mutex

	// Your definitions here.
	// Key value store
	kv_map map[string]string
	resultcache map[int64]Result
}


func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	value, ok := kv.kv_map[args.Key]
	if ok {
		reply.Value = value
	} else {
		reply.Value = ""
	}
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// Check if the request is already processed
	if ok, value := kv.LookupResult(args); ok {
		reply.Value = value
		return
	}

	kv.kv_map[args.Key] = args.Value
	// cache
	kv.resultcache[args.ClientID] = Result{requestId: args.RequestID, value: reply.Value}
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// Check if the request is already processed
	if ok, value := kv.LookupResult(args); ok {
		reply.Value = value
		return
	}

	value, ok := kv.kv_map[args.Key]
	if ok {
		kv.kv_map[args.Key] = value + args.Value
	} else {
		kv.kv_map[args.Key] = args.Value
	}

	// cache
	kv.resultcache[args.ClientID] = Result{requestId: args.RequestID, value: value}
	reply.Value = value
}

func (kv *KVServer) LookupResult(args *PutAppendArgs) (bool, string) {
	value, ok := kv.resultcache[args.ClientID]
	if ok && value.requestId == args.RequestID {
		return true, value.value
	} else {
		return false, ""
	}
}

func StartKVServer() *KVServer {
	kv := new(KVServer)

	// You may need initialization code here.
	kv.kv_map = make(map[string]string)
	kv.resultcache = make(map[int64]Result)

	return kv
}
