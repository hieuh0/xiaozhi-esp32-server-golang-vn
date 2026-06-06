package mqtt_udp

import (
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	UdpSessionStatusActive = "active"
	UdpSessionStatusClosed = "closed"
)

// UdpSession represents a UDP session.
type UdpSession struct {
	ID          string
	Conn        *net.UDPConn //udp conn
	ConnId      string
	ClientId    string
	DeviceId    string
	AesKey      [16]byte // Random 32-digit value.
	Nonce       [8]byte  // Original 16-digit nonce template.
	CreatedAt   time.Time
	LastActive  time.Time
	RemoteAddr  *net.UDPAddr //remote addr
	LocalSeq    uint32
	Block       cipher.Block
	RemoteSeq   uint32
	RecvChannel chan []byte // Audio data to send.
	SendChannel chan []byte // Received audio data.
	Status      string
	Lock        sync.Mutex
}

func (s *UdpSession) SetRemoteAddr(addr *net.UDPAddr) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if addr == nil {
		s.RemoteAddr = nil
		return
	}
	addrCopy := *addr
	s.RemoteAddr = &addrCopy
}

func (s *UdpSession) GetRemoteAddr() *net.UDPAddr {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if s.RemoteAddr == nil {
		return nil
	}
	addrCopy := *s.RemoteAddr
	return &addrCopy
}

func (s *UdpSession) IsClosed() bool {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	return s.Status == UdpSessionStatusClosed
}

func (s *UdpSession) WaitRemoteAddr(timeout time.Duration) *net.UDPAddr {
	if timeout <= 0 {
		return s.GetRemoteAddr()
	}

	deadline := time.Now().Add(timeout)
	for {
		if addr := s.GetRemoteAddr(); addr != nil {
			return addr
		}
		if s.IsClosed() || !time.Now().Before(deadline) {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *UdpSession) DrainPendingAudio() int {
	drained := 0
	for {
		select {
		case <-s.SendChannel:
			drained++
		default:
			return drained
		}
	}
}

// Decrypt decrypts data.
func (s *UdpSession) Decrypt(data []byte) ([]byte, error) {
	// Separate the nonce and ciphertext.
	nonce := data[:16] // Use a 16-byte nonce.
	ciphertext := data[16:]

	// Extract the sequence number.
	seqNum := binary.BigEndian.Uint32(data[12:16])

	// Validate the sequence number.
	/*if seqNum < s.RemoteSeq {
		return nil, fmt.Errorf("sequence number expired: got %d, expected >= %d", seqNum, s.RemoteSeq)
	}*/
	s.RemoteSeq = seqNum

	// Decrypt the data.
	stream := cipher.NewCTR(s.Block, nonce)
	decrypted := make([]byte, len(ciphertext))
	stream.XORKeyStream(decrypted, ciphertext)

	return decrypted, nil
}

// Encrypt encrypts data.
func (s *UdpSession) Encrypt(data []byte) ([]byte, error) {
	// Preallocate memory to avoid growth.
	encrypted := make([]byte, 16+len(data))

	// Build the 16-byte nonce.
	encrypted[0] = 0x01                                          // Packet type.
	binary.BigEndian.PutUint16(encrypted[2:], uint16(len(data))) // Data length.
	copy(encrypted[4:12], s.Nonce[:])                            // Eight-byte nonce.
	s.LocalSeq++
	binary.BigEndian.PutUint32(encrypted[12:], s.LocalSeq) // Sequence number.

	// Encrypt the data.
	stream := cipher.NewCTR(s.Block, encrypted[:16]) // Use 16 bytes as the IV.
	stream.XORKeyStream(encrypted[16:], data)

	return encrypted, nil
}

func (s *UdpSession) GetAesKeyAndNonce() (string, string) {
	// Encode the key and nonce for transport.
	strAesKey := hex.EncodeToString(s.AesKey[:])

	// Build fullNonce: two-byte 0100 prefix + two-byte 0000 length + eight-byte nonce + four-byte 00000000 sequence.
	prefix := []byte{0x01, 0x00}
	length := []byte{0x00, 0x00}
	seq := []byte{0x00, 0x00, 0x00, 0x00}
	fullNonce := append(append(append(prefix, length...), s.Nonce[:]...), seq...)
	strFullNonce := hex.EncodeToString(fullNonce)

	return strAesKey, strFullNonce
}

func (s *UdpSession) RecvData(data []byte) (bool, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if s.Status == UdpSessionStatusClosed {
		return false, nil
	}
	select {
	case s.RecvChannel <- data:
		return true, nil
	default:
		return false, fmt.Errorf("recv channel is full")
	}
}

// SendAudioData sends audio data.
func (s *UdpSession) SendAudioData(data []byte) (bool, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if s.Status == UdpSessionStatusClosed {
		return false, nil
	}
	select {
	case s.SendChannel <- data:
		return true, nil
	default:
		return false, fmt.Errorf("send channel is full")
	}
}

func (s *UdpSession) Destroy() {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if s.Status == UdpSessionStatusClosed {
		return
	}
	s.Status = UdpSessionStatusClosed
	close(s.RecvChannel)
	close(s.SendChannel)
}
