package domain

// Message type constant
const (
	MessageTypeHello  = "hello"  //handshake message
	MessageTypeAbort  = "abort"  //abort message
	MessageTypeListen = "listen" //Listen for messages
	MessageTypeIot    = "iot"    //IoT news
)

// Server message type constants
const (
	ServerMessageTypeHello = "hello" //handshake message
	ServerMessageTypeStt   = "stt"   //speech to text
	ServerMessageTypeTts   = "tts"   //text to speech
	ServerMessageTypeIot   = "iot"   //IoT news
	ServerMessageTypeLlm   = "llm"   //large language model
	ServerMessageTypeText  = "text"  //text message
)

// Message status constant
const (
	MessageStateStart   = "start"   //start state
	MessageStateStop    = "stop"    //stop state
	MessageStateDetect  = "detect"  //detection status
	MessageStateAbort   = "abort"   //abort state
	MessageStateSuccess = "success" //success status
)
