package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// Raft state
const (
	Follower = iota
	Candidate
	Leader
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type LogEntry struct {
	Term int
	Command interface{}
}  

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	currentTerm 	int
	votedFor    	int
	logs         	[]LogEntry

	commitIndex 	int
	lastApplied 	int

	nextIndex   	[]int
	matchIndex  	[]int

	applyMsgChan 	chan ApplyMsg
	applyCond 		*sync.Cond		// condition variable to notify when a new log entry is committed

	state 	 	    int

	// expired timeout
	electionExpired int64
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (3A).
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}


// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}


// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}


// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term 			int
	CandidateId 	int
	LastLogIndex 	int
	LastLogTerm 	int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	voteResult 		bool
	Term 			int
}

//
// 接收到Term更大的RPC消息
//
func (rf *Raft) RecvNewTerm(term int) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.currentTerm = term
	rf.votedFor = -1
	rf.state = Follower

}

func (rf *Raft) NewLeaderEletion(term int, lastLogIndex int, lastLogTerm int) {
	args := &RequestVoteArgs{
		Term: term, 
		CandidateId: rf.me,
		LastLogIndex: lastLogIndex,
		LastLogTerm: lastLogTerm,
	}

	var (
		stop atomic.Bool
		voteCount atomic.Int64
	)
	stop.Store(false)
	voteCount.Store(int64(len(rf.peers) / 2))

	for peer := range rf.peers {
		if stop.Load() {
			break
		}

		if peer == rf.me {
			continue
		}

		// 并发的发送投票请求
		go func(peer int) {
			reply := &RequestVoteReply{}
			ok := rf.sendRequestVote(peer, args, reply)
			if ok {
				// double check 
				// 防止在此期间接受到rpc请求导致状态发生变化
				rf.mu.Lock()
				if rf.state != Candidate || rf.currentTerm != term {
					rf.mu.Unlock()
					return
				}

				if reply.Term > term {
					rf.RecvNewTerm(reply.Term)
					rf.mu.Unlock()
					return
				}

				if reply.voteResult {
					voteCount.Add(-1)
				}

				if voteCount.Load() <= 0 {
					// begin new leader
					rf.state = Leader
					// renew the nextIndex and matchIndex
					for i := range rf.peers {
						rf.nextIndex[i] = len(rf.logs)
						rf.matchIndex[i] = 0
						
						// begin log replication
					}
					stop.Store(true)
					return
				}
			}
		} (peer)


	}
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// callee 的任期比 caller 的大
	if args.Term < rf.currentTerm {
		reply.voteResult = false
		reply.Term = rf.currentTerm
		return
	} else if args.Term > rf.currentTerm {
		rf.RecvNewTerm(args.Term)
	} else {
		// 同一任期，raft node在一个任期只能投给一个node
		if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && (
			(rf.logs[len(rf.logs) - 1].Term < args.LastLogTerm) || (
				rf.logs[len(rf.logs) - 1].Term == args.LastLogTerm && len(rf.logs) - 1 <= args.LastLogIndex)) {
			// vote true
			rf.currentTerm = args.Term
			rf.votedFor = args.CandidateId
			rf.state = Follower

			reply.voteResult = true
			reply.Term = rf.currentTerm
		} else {
			reply.voteResult = false
			reply.Term = rf.currentTerm
		}
	}

}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}


// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).


	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) ticker() {
	for !rf.killed() {
		rf.mu.Lock()

		if ( (rf.state == Follower || rf.state == Candidate) && time.Now().UnixMilli() > rf.electionExpired) {
			// begin new leader selection
			rf.state = Candidate
			rf.currentTerm++
			rf.votedFor = rf.me
			rf.electionExpired = RandTimeStamp()

			var(
				term = rf.currentTerm
				lastLogIndex = len(rf.logs) - 1
				lastLogTerm = rf.logs[lastLogIndex].Term
			)
			rf.mu.Unlock()

			go rf.NewLeaderEletion(term, lastLogIndex, lastLogTerm)

		}

		rf.mu.Unlock()
		// pause for a random amount of time between 50 and 350
		// milliseconds.
		ms := 50 + (rand.Int63() % 300)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (3A, 3B, 3C).
	rf.applyMsgChan = applyCh
	rf.applyCond = sync.NewCond(&rf.mu)

	rf.currentTerm = 0
	rf.votedFor = -1
	
	rf.state = Follower

	rf.logs = make([]LogEntry, 1) // dummy entry at index 0, 具体作用参见论文
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))

	rf.electionExpired = time.Now().UnixMilli() + rand.Int63n(500)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()


	return rf
}

func RandTimeStamp() int64 {
	return time.Now().UnixNano() + rand.Int63n(500)
}