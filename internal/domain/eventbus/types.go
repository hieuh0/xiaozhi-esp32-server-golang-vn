package eventbus

const (
	TopicAddMessage = "add_message"
	TopicSessionEnd = "session_end"
	TopicExitChat   = "exit_chat" //exit chat event

	//Chat history related events (obsolete, use TopicAddMessage uniformly)
	//Deprecated: Use TopicAddMessage instead
	TopicChatHistoryUserMessage      = "chat_history_user_message"      //User messages (post-ASR) - Deprecated
	TopicChatHistoryAssistantMessage = "chat_history_assistant_message" //Robot reply (after LLM+TTS) - Deprecated
)
