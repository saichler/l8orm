package tests

import (
	. "github.com/saichler/l8test/go/infra/t_resources"
	. "github.com/saichler/l8test/go/infra/t_topology"
	"github.com/saichler/layer8/go/overlay/protocol"
	. "github.com/saichler/types/go/common"
)

var topo *TestTopology

func init() {
	Log.SetLogLevel(Trace_Level)
	protocol.UsingContainers = false
}

func setup() {
	setupTopology()
}

func tear() {
	shutdownTopology()
}

func reset(name string) {
	Log.Info("*** ", name, " end ***")
	topo.ResetHandlers()
}

func setupTopology() {
	topo = NewTestTopology(4, 20000, 30000, 40000)
}

func shutdownTopology() {
	topo.Shutdown()
}
