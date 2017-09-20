package Instance

import (
	"rtmpmate.com/events"
	"rtmpmate.com/muxer"
	"rtmpmate.com/muxer/DASHMuxer"
	"rtmpmate.com/muxer/FLVMuxer"
	"rtmpmate.com/muxer/FMP4Muxer"
	"rtmpmate.com/muxer/HLSMuxer"
	"rtmpmate.com/net/rtmp/NetConnection"
	StreamTypes "rtmpmate.com/net/rtmp/Stream/Types"
	"sync"
	"syscall"
)

type Instance struct {
	Name string

	clients    map[string]*NetConnection.NetConnection
	clientsMtx sync.RWMutex
	streams    map[string]*Instream
	streamsMtx sync.RWMutex

	statsToAdmin
	events.EventDispatcher
}

type stats struct {
	bytesIn  int
	bytesOut int

	msgIn      int
	msgOut     int
	msgDropped int
}

type statsToAdmin struct {
	stats

	launchTime uint64 // time the application started.
	upTime     uint64 // time, in seconds, the application has been running.

	totalConnects    int // Total number of socket connections to this instance since the instance was started.
	totalDisconnects int // Total number of socket disconnections from this instance since the instance was started.

	accepted        int // Total number of connection attempts accepted by this instance.
	rejected        int // Total number of connection attempts rejected by this instance.
	connected       int // Total number of connections currently active.
	normalConnects  int // Total number of normal connections.
	virtualConnects int // Total number of connections through a remote edge.
	adminConnects   int // Total number of administrator connections.
	debugConnects   int // Total number of debug connections.

	pid   int  // The pid of the core process running the instance.
	debug bool // true if a debug session is enabled, otherwise false.

	swfVerificationAttempts int
	swfVerificationMatches  int
	swfVerificationFailures int
}

type Instream struct {
	Name      string
	Type      string
	Muxer     *muxer.Muxer
	FLVMuxer  *FLVMuxer.FLVMuxer
	HLSMuxer  *HLSMuxer.HLSMuxer
	FMP4Muxer *FMP4Muxer.FMP4Muxer
	DASHMuxer *DASHMuxer.DASHMuxer
}

func New(name string) (*Instance, error) {
	if name == "" {
		name = "_definst_"
	}

	var inst Instance
	inst.Name = name
	inst.clients = make(map[string]*NetConnection.NetConnection)
	inst.streams = make(map[string]*Instream)

	return &inst, nil
}

func (this *Instance) GetStream(name string) (*Instream, error) {
	if name == "" {
		return nil, syscall.EINVAL
	}

	this.streamsMtx.Lock()

	s, ok := this.streams[name]
	if ok == false {
		var ins Instream
		ins.Name = name
		ins.Type = StreamTypes.IDLE
		ins.Muxer, _ = muxer.New()
		ins.FLVMuxer, _ = FLVMuxer.New()
		ins.HLSMuxer, _ = HLSMuxer.New()
		ins.FMP4Muxer, _ = FMP4Muxer.New()
		ins.DASHMuxer, _ = DASHMuxer.New()

		s = &ins
		this.streams[name] = s
	}

	this.streamsMtx.Unlock()

	return s, nil
}

func (this *Instance) Unload() {
	this.clientsMtx.Lock()
	for _, nc := range this.clients {
		nc.Close()
		delete(this.clients, nc.FarID)
	}
	this.clientsMtx.Unlock()

	explain := "unloading instance"
	this.streamsMtx.Lock()
	for _, s := range this.streams {
		s.Muxer.EndOfStream(explain)
		s.FLVMuxer.EndOfStream(explain)
		s.HLSMuxer.EndOfStream(explain)
		s.FMP4Muxer.EndOfStream(explain)
		s.DASHMuxer.EndOfStream(explain)
		delete(this.streams, s.Name)
	}
	this.streamsMtx.Unlock()
}

func (this *Instance) GetStats() *stats {
	return &this.stats
}

func (this *Instance) Add(nc *NetConnection.NetConnection) {
	this.clientsMtx.Lock()

	this.clients[nc.FarID] = nc
	this.connected++

	this.clientsMtx.Unlock()
}

func (this *Instance) Remove(nc *NetConnection.NetConnection) {
	this.clientsMtx.Lock()

	delete(this.clients, nc.FarID)
	this.connected--

	this.clientsMtx.Unlock()
}
