package net

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Heliodex/litecode/keys"
)

const FindStart = "cofind:"

func PeerFromFindString(find string) (p keys.Peer, err error) {
	if !strings.HasPrefix(find, FindStart) || find[56] != '.' {
		return keys.Peer{}, fmt.Errorf("not a valid find string")
	}

	pk, err := keys.DecodePK(keys.PubStart + find[7:56]) // up until 1st dot
	if err != nil {
		return
	}

	rest := find[57:] // after the dot

	decodedAddrs, err := base64.RawURLEncoding.DecodeString(rest)
	if err != nil {
		return
	}

	addrs, ok := pk.Verify(decodedAddrs)
	if !ok {
		return keys.Peer{}, fmt.Errorf("invalid addresses signature")
	} else if len(addrs)%keys.AddressLen != 0 {
		return keys.Peer{}, fmt.Errorf("invalid addresses part length")
	}

	addresses := make([]keys.Address, len(addrs)/keys.AddressLen)
	for i := range addresses {
		copy(addresses[i][:], addrs[i*keys.AddressLen:][:keys.AddressLen])
	}

	return keys.Peer{Pk: pk, Addresses: addresses}, nil
}

type (
	EncryptedMessage []byte
	MessageType      uint8
)

const (
	Msg1 = iota
)

type Message struct {
	From keys.Peer
	Type MessageType
	Body []byte
}

func (m EncryptedMessage) Decode(kp keys.Keypair) (msg Message, err error) {
	from, body, err := kp.Decrypt(m)
	if err != nil {
		return
	}

	return Message{
		From: from,
		Type: MessageType(body[0]),
		Body: body[1:],
	}, nil
}

type Node struct {
	keys.ThisPeer

	Peers      []keys.Peer // known peers
	SendRaw    func(peer keys.Peer, msg []byte) (err error)
	ReceiveRaw <-chan EncryptedMessage
}

func (n *Node) Send(p keys.Peer, t MessageType, msg []byte) (err error) {
	msg = append([]byte{byte(t)}, msg...)

	ct, err := n.ThisPeer.Encrypt(msg, p.Pk)
	if err != nil {
		return
	}

	return n.SendRaw(p, ct)
}

// A find string encodes the pk and addresses
func (n *Node) FindString() string {
	pk := n.Kp.Pk.Encode()[6:]

	addrs := make([]byte, len(n.Addresses)*keys.AddressLen)
	for i, addr := range n.Addresses {
		copy(addrs[i*keys.AddressLen:], addr[:])
	}

	signedAddrs := n.Kp.Sk.Sign(addrs)                                // yes, actually works now
	encodedAddrs := base64.RawURLEncoding.EncodeToString(signedAddrs) // we might do ipv6/libp2p/port enocding or smth later
	return fmt.Sprintf("cofind:%s.%s", pk, encodedAddrs)
}

// unoptimised; debug
func (n *Node) log(msg ...any) {
	pke := n.Kp.Pk.Encode()
	logId := pke[6:8]

	m := strings.ReplaceAll(fmt.Sprint(msg...), "\n", "\n     ")
	fmt.Printf("[%s]\n     %s\n", logId, m)
}

func (n *Node) HandleMessage(msg Message) {
	switch msg.Type {
	case Msg1:
		n.Send(msg.From, Msg1, []byte("sup")) // infinite loop can't be avoided if you take a route straight through what is known as
	}
}

func (n *Node) Start() {
	pke := n.Kp.Pk.Encode()

	n.log(
		"Starting\n",
		"I'm ", pke, "\n",
		"My primary address is ", n.Addresses[0], "\n",
		"I know ", len(n.Peers), " peers")

	// Receiver
	go func() {
		for {
			rec := <-n.ReceiveRaw
			msg, err := rec.Decode(n.Kp)
			if err != nil {
				n.log("Failed to decode message\n", err)
				continue
			}

			n.log(
				"Received ", len(msg.Body), "\n",
				"From ", msg.From.Pk.Encode(), "\n",
				"@ ", msg.From.Addresses[0], "\n")

			n.HandleMessage(msg)
		}
	}()

	// Sender
	for _, peer := range n.Peers {
		n.log(
			"To   ", peer.Pk.Encode(), "\n",
			"@ ", peer.Addresses[0])

		n.Send(peer, Msg1, []byte("Hello, peer!"))
	}
}
