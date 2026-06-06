package mqtt_udp

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	. "xiaozhi-esp32-server-golang/logger"
)

// UdpServer manages UDP sessions and packet transport.
/*
type UDPServer struct {
	conn       *net.UDPConn
	sessions   map[string]*Session
	mqttServer *MqttServer
	udpPort    int
	sync.RWMutex
}*/

type UdpServer struct {
	conn           *net.UDPConn
	udpPort        int      //udp server listen port
	externalHost   string   //udp server external host
	externalPort   int      //udp server external port
	connId2Session sync.Map //connId => UdpSession
	mqttAdapter    *MqttUdpAdapter
	sync.RWMutex
}

const maxConnIDGenerateAttempts = 16

var udpRandReader io.Reader = rand.Reader

// NewUDPServer creates a UDP server.
func NewUDPServer(udpPort int, externalHost string, externalPort int) *UdpServer {
	return &UdpServer{
		udpPort:        udpPort,
		externalHost:   externalHost,
		externalPort:   externalPort,
		connId2Session: sync.Map{},
	}
}

// Start starts the UDP server.
func (s *UdpServer) Start() error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: s.udpPort,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %v", err)
	}

	s.conn = conn
	Infof("UDP server started on %s:%d", "0.0.0.0", s.udpPort)

	// Start session cleanup.
	//go s.cleanupSessions()

	// Start packet processing.
	go s.handlePackets()

	return nil
}

// Close shuts down the UDP server and causes handlePackets to exit.
func (s *UdpServer) Close() error {
	s.Lock()
	conn := s.conn
	s.conn = nil
	s.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close()
}

// handlePackets processes received packets.
func (s *UdpServer) handlePackets() {
	buffer := make([]byte, 4096) // Use the default buffer size.
	for {
		s.RLock()
		conn := s.conn
		s.RUnlock()
		if conn == nil {
			return
		}
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			s.RLock()
			closed := s.conn == nil
			s.RUnlock()
			if closed {
				return
			}
			Errorf("failed to read UDP data: %v", err)
			continue
		}

		// Copy the data to avoid concurrent modification.
		data := make([]byte, n)
		copy(data, buffer[:n])

		// Process the packet.
		s.processPacket(addr, data)
	}
}

func (s *UdpServer) getSessionByConnID(connID string) *UdpSession {
	val, ok := s.connId2Session.Load(connID)
	if ok {
		return val.(*UdpSession)
	}
	return nil
}

// processPacket processes a single packet.
func (s *UdpServer) processPacket(addr *net.UDPAddr, data []byte) {
	// Validate the packet size.
	if len(data) < 16 {
		Warn("packet is too small")
		return
	}

	fullNonce := data[:16]
	connID := fullNonce[4:8] // Use bytes 5-8 as the connection ID.
	strConnID := hex.EncodeToString(connID)
	udpSession := s.getSessionByConnID(strConnID)
	if udpSession == nil {
		//Warnf("session does not exist addr: %s, connID: %s", addr, strConnID)
		return
	}

	// Update the last activity time.
	udpSession.LastActive = time.Now()

	decrypted, err := udpSession.Decrypt(data)
	if err != nil {
		Errorf("addr: %s decryption failed: %v", addr, err)
		return
	}
	currentAddr := udpSession.GetRemoteAddr()
	if currentAddr == nil || currentAddr.String() != addr.String() {
		udpSession.SetRemoteAddr(addr)
	}
	Debugf("received audio data, addr: %s, size: %d bytes", addr, len(decrypted))
	ok, err := udpSession.RecvData(decrypted)
	if err != nil {
		Errorf("addr: %s failed to receive data: %v", addr, err)
		return
	}
	if !ok {
		Warnf("addr: %s failed to receive data, channel is full", addr)
		return
	}
	/*select {
	case udpSession.RecvChannel <- decrypted:
		return
	default:
		Warnf("udpSession.RecvChannel is full, addr: %s", addr)
	}*/
}

// cleanupSessions removes expired sessions.
func (s *UdpServer) cleanupSessions() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		now := time.Now()
		s.connId2Session.Range(func(key, value interface{}) bool {
			session := value.(*UdpSession)
			if now.Sub(session.LastActive) > 5*time.Minute {
				s.connId2Session.Delete(key)
				Infof("removed expired session: %s", key)
			}
			return true
		})
	}
}

// CreateSession creates a session.
func (s *UdpServer) CreateSession(deviceId, clientId string) *UdpSession {
	// Generate a session ID.
	sessionID, err := generateSessionID()
	if err != nil {
		Errorf("failed to generate session ID: %v", err)
		return nil
	}

	// Generate an AES key.
	key := make([]byte, 16)
	if err := fillRandomBytes(key); err != nil {
		Errorf("failed to generate AES key: %v", err)
		return nil
	}

	// Create the AES block cipher.
	block, err := aes.NewCipher(key)
	if err != nil {
		Errorf("failed to create AES block cipher: %v", err)
		return nil
	}

	// Convert the key to [16]byte.
	aesKey := [16]byte{}
	copy(aesKey[:], key)

	for attempt := 0; attempt < maxConnIDGenerateAttempts; attempt++ {
		// Generate a four-byte connection ID.
		connID := make([]byte, 4)
		if err := fillRandomBytes(connID); err != nil {
			Errorf("failed to generate connection ID: %v", err)
			return nil
		}
		strConnID := hex.EncodeToString(connID)

		// Four-byte timestamp.
		timestamp := make([]byte, 4)
		binary.BigEndian.PutUint32(timestamp, uint32(time.Now().Unix()))

		// Build the nonce from the four-byte connection ID and four-byte timestamp.
		nonce := append(connID, timestamp...)

		// Convert the nonce to [8]byte.
		nonceBytes := [8]byte{}
		copy(nonceBytes[:], nonce)

		// Create the session.
		session := &UdpSession{
			ID:          sessionID,
			ConnId:      strConnID,
			ClientId:    clientId,
			DeviceId:    deviceId,
			AesKey:      aesKey,
			Nonce:       nonceBytes, // Preserve the original nonce template.
			CreatedAt:   time.Now(),
			LastActive:  time.Now(),
			Block:       block,
			RecvChannel: make(chan []byte, 100),
			SendChannel: make(chan []byte, 100),
			Status:      UdpSessionStatusActive,
			Lock:        sync.Mutex{},
		}

		if _, loaded := s.connId2Session.LoadOrStore(strConnID, session); loaded {
			Warnf("UDP connID collision; retrying generation: device=%s, connID=%s, attempt=%d", deviceId, strConnID, attempt+1)
			continue
		}

		s.startSessionSender(session)
		return session
	}

	Errorf("failed to generate a unique UDP connID: device=%s", deviceId)
	return nil
}

func (s *UdpServer) startSessionSender(session *UdpSession) {
	go func() {
		for data := range session.SendChannel {
			remoteAddr := session.WaitRemoteAddr(2 * time.Second)
			if remoteAddr == nil {
				dropped := 1 + session.DrainPendingAudio()
				Warnf("UDP remote address is not established; dropping TTS audio: device=%s, connId=%s, dropped=%d", session.DeviceId, session.ConnId, dropped)
				continue
			}
			encrypted, err := session.Encrypt(data)
			if err != nil {
				Errorf("encryption failed: %v", err)
				continue
			}
			_, err = s.writeToUDP(encrypted, remoteAddr)
			if err != nil {
				Errorf("failed to send audio data: %v", err)
				continue
			}
		}
	}()
}

func (s *UdpServer) writeToUDP(data []byte, remoteAddr *net.UDPAddr) (int, error) {
	s.RLock()
	conn := s.conn
	s.RUnlock()
	if conn == nil {
		return 0, fmt.Errorf("udp server is closed")
	}
	return conn.WriteToUDP(data, remoteAddr)
}

// CloseSession closes a session.
func (s *UdpServer) CloseSession(connID string) {
	session := s.getSessionByConnID(connID)
	s.CloseSessionByRef(session)
}

// ClearSessionAddrBinding clears the session's UDP address binding without destroying the session.
func (s *UdpServer) ClearSessionAddrBinding(connID string) {
	session := s.getSessionByConnID(connID)
	if session == nil {
		return
	}
	session.SetRemoteAddr(nil)
}

func (s *UdpServer) SetConnId2Session(connID string, session *UdpSession) {
	Debugf("SetConnId2Session, connID: %s, session: %+v", connID, session)
	s.connId2Session.Store(connID, session)
}

// GetSessionByConnID returns a session by connection ID.
func (s *UdpServer) GetSessionByConnID(connID string) *UdpSession {
	val, ok := s.connId2Session.Load(connID)
	if ok {
		return val.(*UdpSession)
	}
	return nil
}

// generateSessionID generates a session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 8)
	if err := fillRandomBytes(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func fillRandomBytes(buffer []byte) error {
	_, err := io.ReadFull(udpRandReader, buffer)
	return err
}

func (s *UdpServer) CloseSessionByRef(session *UdpSession) {
	if session == nil {
		return
	}
	s.connId2Session.Delete(session.ConnId)
	session.SetRemoteAddr(nil)
	session.Destroy()
}
