package hashinfo

import (
	"net"
	"github.com/yqsy/recipes/dht/helpful"
	"github.com/yqsy/recipes/dht/transaction"
	"github.com/yqsy/recipes/dht/hashinfocommon"
	"github.com/yqsy/recipes/dht/inspector"
	"github.com/yqsy/recipes/dht/metadata"
	"strconv"
	"github.com/op/go-logging"
	"github.com/yqsy/recipes/dht/flowcontrol"
	"time"
	"github.com/yqsy/recipes/dht/bencode"
)

var log = logging.MustGetLogger("dht")

type HashInfoGetter struct {
	dhtNodes []string

	// randmon 20 bytes id
	selfId string

	localAddr string

	// send req, get res
	// or get req, reply res
	// all use this udp conn
	serverConn *net.UDPConn

	// gen id
	tm transaction.Transaction

	// res prototype pool
	// one req match unique res
	resPrototypeDict map[string]interface{}

	// for join unique check
	uniqueNodePool map[string]struct{}

	// output
	MetaSourceChan chan *metadata.MetaSource

	// for monitor
	Ins *inspector.Inspector

	FlowControl *flowcontrol.FlowControl
}

type UdpMsg struct {
	Packet     []byte
	RemoteAddr *net.UDPAddr
}

func NewHashInfoGetter(ins *inspector.Inspector) *HashInfoGetter {
	hg := &HashInfoGetter{}
	hg.dhtNodes = []string{
		"router.bittorrent.com:6881",
		"router.utorrent.com:6881",
		"dht.transmissionbt.com:6881"}
	hg.selfId = helpful.RandomString(20)
	hg.localAddr = ":6881"
	hg.resPrototypeDict = make(map[string]interface{})
	hg.uniqueNodePool = make(map[string]struct{})
	hg.MetaSourceChan = make(chan *metadata.MetaSource, 1024)
	hg.Ins = ins
	hg.FlowControl = flowcontrol.NewFlowControl()

	hg.Ins.SafeDo(func() {
		hg.Ins.BasicNodes = hg.dhtNodes
		hg.Ins.SelfId = hg.selfId
		hg.Ins.LocalAddr = hg.localAddr
	})

	return hg
}

func (hg *HashInfoGetter) Run() error {
	if serverAddr, err := net.ResolveUDPAddr("udp", hg.localAddr); err != nil {
		return err
	} else {
		if hg.serverConn, err = net.ListenUDP("udp", serverAddr); err != nil {
			return err
		} else {
			if err := hg.SendJoin(); err != nil {
				panic(err)
			}
		}
	}

	go func() {
		hg.FlowControl.Increasing(512)
	}()

	produceUdpMsgChan := make(chan *UdpMsg)
	consumeOkChan := make(chan struct{})

	go func() {
		hg.ProducingUdpMsg(consumeOkChan, produceUdpMsgChan)
	}()

	rejoinTicker := time.NewTicker(time.Second * 5)

	consumeOkChan <- struct{}{} // start
	for {
		select {
		case udpMsg := <-produceUdpMsgChan:
			hg.DispatchReqAndRes(udpMsg.Packet, udpMsg.RemoteAddr)
			consumeOkChan <- struct{}{} // cousume this msg ok. begin to receive next one
		case <-rejoinTicker.C:
			if err := hg.SendJoin(); err != nil {
				log.Warningf("send join err: %v", err)
			}
		}
	}
}

func (hg *HashInfoGetter) ProducingUdpMsg(consumOkChan chan struct{}, produceUdpMsgChan chan *UdpMsg) {
	buf := make([]byte, 2048)
	for {
		<-consumOkChan

		if rn, remoteAddr, err := hg.serverConn.ReadFromUDP(buf); err != nil {
			panic(err)
		} else {
			udpMsg := &UdpMsg{Packet: buf[:rn], RemoteAddr: remoteAddr}

			produceUdpMsgChan <- udpMsg
		}
	}
}

func (hg *HashInfoGetter) DispatchReqAndRes(buf []byte, remoteAddr *net.UDPAddr) {
	if b, err := bencode.Decode(string(buf)); err != nil {
		// log.Warningf("decode err: %v from: %v", err, remoteAddr)
	} else {
		obj, err := hashinfocommon.GetObjWithCheck(b)
		if err != nil {
			//log.Warningf("err msg: %v from: %v", err, remoteAddr)
			return
		}

		switch obj["y"].(string) {
		case "q":
			hg.DispatchReq(obj, remoteAddr)
		case "r":
			hg.DispatchRes(obj)
		case "e":
			hg.DispatchError(obj)
		default:
			log.Warningf("error \"y\": %v from: %v", obj["y"].(string), remoteAddr)
		}
	}
}

func (hg *HashInfoGetter) SendReq(reqBytes []byte, remoteAddr *net.UDPAddr, tid string, resType string) error {
	if _, err := hg.serverConn.WriteToUDP(reqBytes, remoteAddr); err != nil {
		return err
	} else {
		hg.resPrototypeDict[tid] = resType
		hg.Ins.SafeDo(func() {
			hg.Ins.UnReplyTid[tid] = struct{}{}
		})
		return nil
	}
}

func (hg *HashInfoGetter) SendJoin() error {
	for _, nodeAddr := range hg.dhtNodes {
		if err := hg.SendFindNode(nodeAddr, hg.selfId, hg.selfId); err != nil {
			return err
		}
	}
	return nil
}

func (hg *HashInfoGetter) SendFindNode(nodeAddr string, selfId, targetId string) error {
	tid := hg.tm.FetchAndAdd()

	reqFindNodes := map[string]interface{}{
		"t": tid,
		"y": "q",
		"q": "find_node",
		"a": map[string]interface{}{
			"id":     selfId,
			"target": targetId,
		},
	}

	reqBytes := []byte(bencode.Encode(reqFindNodes))

	if nodeAddr, err := net.ResolveUDPAddr("udp", nodeAddr); err != nil {
		return err
	} else {
		if err = hg.SendReq(reqBytes, nodeAddr, tid, "find_node"); err != nil {
			return err
		}
		hg.Ins.SafeDo(func() {
			hg.Ins.SendedFindNodeNumber += 1
		})
	}

	return nil
}

func (hg *HashInfoGetter) DispatchReq(req map[string]interface{}, remoteAddr *net.UDPAddr) {
	switch req["q"].(string) {
	case "ping":
		hg.Ins.SafeDo(func() {
			hg.Ins.ReceivedPingNumber += 1
		})
		hg.HandleReqPing(req, remoteAddr)
	case "find_node":
		hg.Ins.SafeDo(func() {
			hg.Ins.ReceivedFindNodeNumber += 1
		})
		hg.HandleReqFindNode(req, remoteAddr)
	case "get_peers":
		hg.Ins.SafeDo(func() {
			hg.Ins.ReceivedGetPeersNumber += 1
		})
		hg.HandleReqGetPeers(req, remoteAddr)
	case "announce_peer":
		hg.Ins.SafeDo(func() {
			hg.Ins.ReceivedGetAnnouncePeerNumber += 1
		})
		hg.HandleReqAnnouncePeer(req, remoteAddr)
	default:
		//log.Warningf("unknown req type: %v", req["q"].(string))
	}
}

func (hg *HashInfoGetter) HandleReqPing(req map[string]interface{}, remoteAddr *net.UDPAddr) {
	if err := hashinfocommon.CheckReqPingValid(req); err != nil {
		log.Warningf("not valid ReqPing err: %v", err)
	} else {
		resPing := map[string]interface{}{
			"t": req["t"].(string),
			"y": "r",
			"r": map[string]interface{}{
				"id": req["a"].(map[string]interface{})["id"].(string),
			},
		}

		resBytes := []byte(bencode.Encode(resPing))
		if _, err = hg.serverConn.WriteToUDP(resBytes, remoteAddr); err != nil {
			log.Warningf("write udp err: %v", err)
		}
	}
}

func (hg *HashInfoGetter) HandleReqFindNode(req map[string]interface{}, remoteAddr *net.UDPAddr) {
	resFindNode := map[string]interface{}{
		"t": req["t"].(string),
		"y": "r",
		"r": map[string]interface{}{
			"id":    hg.selfId,
			"nodes": "",
		},
	}

	resBytes := []byte(bencode.Encode(resFindNode))
	if _, err := hg.serverConn.WriteToUDP(resBytes, remoteAddr); err != nil {
		log.Warningf("write udp err: %v", err)
	}

}

func (hg *HashInfoGetter) HandleReqGetPeers(req map[string]interface{}, remoteAddr *net.UDPAddr) {
	// TODO what is token?

	if err := hashinfocommon.CheckReqGetPeersValid(req); err != nil {
		log.Warningf("not valid HandleReqGetPeers err: %v", err)
	} else {
		token := req["a"].(map[string]interface{})["info_hash"].(string)[:2]

		resGetPeers := map[string]interface{}{
			"t": req["t"].(string),
			"y": "r",
			"r": map[string]interface{}{
				"id":    hg.selfId,
				"token": token,
				"nodes": "",
			},
		}

		resBytes := []byte(bencode.Encode(resGetPeers))
		if _, err := hg.serverConn.WriteToUDP(resBytes, remoteAddr); err != nil {
			log.Warningf("write udp err: %v", err)
		}
	}
}

func (hg *HashInfoGetter) HandleReqAnnouncePeer(req map[string]interface{}, remoteAddr *net.UDPAddr) {
	if err := hashinfocommon.CheckReqAnnouncePeerValid(req); err != nil {
		log.Warningf("not valid ReqAnnouncePeer err: %v", err)
	} else {
		a := req["a"].(map[string]interface{})

		// If it is present and non-zero, the port argument should be ignored and the source port of
		// the UDP packet should be used as the peer's port instead.
		port := strconv.Itoa(a["port"].(int))

		if impliedPort, ok := a["implied_port"]; ok {
			if impliedPort, ok := impliedPort.(int); ok && impliedPort != 0 {
				port = strconv.Itoa(remoteAddr.Port)
			}
		}

		peerAddr := remoteAddr.IP.String() + ":" + port

		hg.MetaSourceChan <- &metadata.MetaSource{
			Infohash: a["info_hash"].(string),
			Addr:     peerAddr}

		resAnnouncePeer := map[string]interface{}{
			"t": req["t"].(string),
			"y": "r",
			"r": map[string]interface{}{
				"id": hg.selfId,
			},
		}

		resBytes := []byte(bencode.Encode(resAnnouncePeer))
		if _, err := hg.serverConn.WriteToUDP(resBytes, remoteAddr); err != nil {
			log.Warningf("write udp err: %v", err)
		}
	}

}

func (hg *HashInfoGetter) DispatchRes(res map[string]interface{}) {
	tid := res["t"].(string)
	if prototype, ok := hg.resPrototypeDict[tid]; ok {
		delete(hg.resPrototypeDict, tid)
		hg.Ins.SafeDo(func() {
			delete(hg.Ins.UnReplyTid, tid)
		})

		switch prototype {
		case "find_node":
			hg.HandleResFindNode(res)
		default:
			panic("impossible")
		}

	} else {
		//log.Warningf("not match res received tid: %v,drop it", helpful.Get10Hex(tid))
	}
}

func (hg *HashInfoGetter) HandleResFindNode(res map[string]interface{}) {
	if err := hashinfocommon.CheckResFindNodeValid(res); err != nil {
		//log.Warningf("not valid ResFindNode err: %v", err)
	} else {
		r := res["r"].(map[string]interface{})
		nodes := hashinfocommon.GetNodes(r["nodes"].(string))
		for _, node := range nodes {
			if _, ok := hg.uniqueNodePool[node.Id]; ok {
				continue
			}
			hg.uniqueNodePool[node.Id] = struct{}{}

			hg.FlowControl.WaitFlow()

			if err := hg.SendFindNode(node.Addr, node.Id[15:]+hg.selfId[:15], node.Id); err != nil {
				log.Warningf("send find_node err: %v", err)
			}
		}
	}
}

func (hg *HashInfoGetter) DispatchError(err map[string]interface{}) {
	tid := err["t"].(string)
	if _, ok := hg.resPrototypeDict[tid]; ok {
		delete(hg.resPrototypeDict, tid)

		hg.Ins.SafeDo(func() {
			hg.Ins.ReceivedErrors += 1
		})
		code := err["e"].([]interface{})[0].(int)
		description := err["e"].([]interface{})[1].(string)

		_ = code
		_ = description
		//log.Warningf("received a err: %v %v , tid: %v", code, description, helpful.Get10Hex(tid))
	} else {
		//log.Warningf("received a tid not match res, tid: %v", helpful.Get10Hex(tid))
	}
}
