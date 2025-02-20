package net

import "github.com/Heliodex/litecode/keys"

// real secret keys for the purposes of testing
var sampleKeys = [...]string{
	"cosec:0aqouiilz3-ynmmxunwx1-7u6e5xppqa-hmz7q8yd3f-5l92e17yos",
	"cosec:0ot4jpb8z4-iq7yu96m3f-9bh2ze9s7w-m7r7vowu2k-tl8pmbetoz",
	"cosec:50u4onk3m0-owyszhfou0-5uvrymlofu-brye4mkomo-3vr2cta2sa",
	"cosec:1omi5wd5ry-acq82a36oo-d73ls1y7h8-tna64ml180-gb4cxjpgk4",
	"cosec:1nikowcxso-yaxz7ewktj-n4cj0bklsd-xbdsl2ipaw-91vww4cex4",
	"cosec:3a1r7x85ki-duan0b0wlk-ate5tun2ag-mdmk5kghrc-3rcpir16w6",
	"cosec:08al1krxnf-u0kmgplotd-yr7fatryv8-9ktqeba3xz-xmzwviykjc",
}
var sampleKeysUsed uint8 = 0

func getSampleKeypair() (kp keys.Keypair) {
	sampleKeysUsed++
	if sampleKeysUsed > uint8(len(sampleKeys)) {
		panic("no more sample keys")
	} else if skBytes, err := keys.DecodeSK(sampleKeys[sampleKeysUsed-1]); err != nil {
		panic("invalid sample key")
	} else if kp, err = keys.KeypairSK(skBytes); err != nil {
		panic("invalid keypair")
	}

	return
}

type LocalPeer struct {
	Peer
	Receive chan<- Message
}

type LocalNet struct {
	ExistingPeers []LocalPeer
}

func (n *LocalNet) AddPeer(p Peer, recv chan<- Message) {
	n.ExistingPeers = append(n.ExistingPeers, LocalPeer{p, recv})
}

func (n LocalNet) Send(p Peer, m Message) {
	for _, ep := range n.ExistingPeers {
		if p.Equals(ep.Peer) {
			ep.Receive <- m
			return
		}
	}
}

func (n *LocalNet) NewNode() (node Node) {
	kp := getSampleKeypair()
	peer := Peer{
		kp.Pk,
		[]Address{{sampleKeysUsed}}, // sequential placeholder
	}

	recv := make(chan Message)
	n.AddPeer(peer, recv)
	node = Node{
		Peer:    peer,
		Kp:      kp,
		Send:    n.Send,
		Receive: recv,
	}

	go node.Start()
	return
}
