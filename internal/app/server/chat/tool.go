package chat

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	data_client "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	mcp_go "github.com/mark3labs/mcp-go/mcp"
)

type toolCallResponseSummary struct {
	invokeToolSuccess bool
	hasMediaOutput    bool
}

// handleToolCallResponse handles the tool call response
func (l *LLMManager) handleToolCallResponse(ctx context.Context, respMsg *schema.Message, tools []schema.ToolCall, executor *toolCallExecutor) (toolCallResponseSummary, error) {
	if len(tools) == 0 {
		return toolCallResponseSummary{}, nil
	}

	tools = normalizeToolCallIDs(tools)
	if respMsg != nil && len(respMsg.ToolCalls) > 0 {
		respMsg.ToolCalls = normalizeToolCallIDs(respMsg.ToolCalls)
	}

	log.Infof("Handled %d tool calls", len(tools))
	if executor == nil {
		executor = newToolCallExecutor(l, ctx)
		executor.Submit(tools)
	}

	var messageList []*schema.Message

	// Only add to messageList when respMsg has content (Content is not empty or ToolCalls is not empty)
	// Avoid saving empty assistant messages causing 400 errors in subsequent LLM calls
	if respMsg != nil && (respMsg.Content != "" || len(respMsg.ToolCalls) > 0) {
		messageList = append(messageList, respMsg)
	}

	invokeToolSuccess := false
	findExitTool := false
	shouldStopLLMProcessing := false
	hasMediaOutput := false

	results := executor.Wait()
	for _, result := range results {
		if result.invokeToolSuccess {
			invokeToolSuccess = true
		}
		if result.findExitTool {
			findExitTool = true
		}
		if result.shouldStopLLMProcessing {
			shouldStopLLMProcessing = true
		}
		if result.hasMediaOutput {
			hasMediaOutput = true
		}
		messageList = append(messageList, result.message)
	}

	if len(messageList) > 0 {
		for _, msg := range messageList {
			// Filter out assistant messages with empty Content to avoid saving them in the history
			// Empty assistant messages will cause 400 errors on subsequent LLM calls
			if msg != nil && msg.Role == schema.Assistant && msg.Content == "" && len(msg.ToolCalls) == 0 {
				log.Debugf("Skip saving empty assistant messages")
				continue
			}
			l.AddLlmMessage(ctx, msg)
		}
	}

	if l.clientState.GetMemoryMode() == data_client.MemoryModeNone {
		ctx = appendToolRoundMessagesToContext(ctx, messageList)
	}

	executor.WaitMedia()

	if findExitTool {
		// Post exit chat event
		eventbus.Get().Publish(eventbus.TopicExitChat, &eventbus.ExitChatEvent{
			ClientState: l.clientState,
			Reason:      "Tool call exit",
			TriggerType: "tool_call",
			UserText:    "",
			Timestamp:   time.Now(),
		})

		return toolCallResponseSummary{
			invokeToolSuccess: invokeToolSuccess,
			hasMediaOutput:    hasMediaOutput,
		}, nil
	}

	// If the tool call is successful and is not marked to stop processing, continue the LLM call
	if invokeToolSuccess && !shouldStopLLMProcessing {
		l.DoLLmRequest(ctx, nil, l.einoTools, true, nil)
	}

	return toolCallResponseSummary{
		invokeToolSuccess: invokeToolSuccess,
		hasMediaOutput:    hasMediaOutput,
	}, nil
}

type toolCallExecutionResult struct {
	order                   int
	message                 *schema.Message
	invokeToolSuccess       bool
	findExitTool            bool
	shouldStopLLMProcessing bool
	hasMediaOutput          bool
}

type toolCallExecutor struct {
	manager *LLMManager
	ctx     context.Context
	toolCtx context.Context

	mu             sync.Mutex
	submittedCall  map[string]struct{}
	submitSequence int
	results        []toolCallExecutionResult

	wg      sync.WaitGroup
	mediaWg sync.WaitGroup
}

func newToolCallExecutor(manager *LLMManager, ctx context.Context) *toolCallExecutor {
	toolCtx := ctx
	if chatSessionOperator, ok := ctx.Value("chat_session_operator").(ChatSessionOperator); ok {
		toolCtx = context.WithValue(ctx, "chat_session_operator", chatSessionOperator)
	}

	return &toolCallExecutor{
		manager:       manager,
		ctx:           ctx,
		toolCtx:       toolCtx,
		submittedCall: make(map[string]struct{}),
	}
}

func (e *toolCallExecutor) Submit(toolCalls []schema.ToolCall) {
	for _, tc := range toolCalls {
		toolCall := ensureToolCallID(tc)
		callID := toolCall.ID

		e.mu.Lock()
		if _, exists := e.submittedCall[callID]; exists {
			e.mu.Unlock()
			continue
		}
		e.submittedCall[callID] = struct{}{}
		order := e.submitSequence
		e.submitSequence++
		e.wg.Add(1)
		e.mu.Unlock()

		go func() {
			defer e.wg.Done()
			result := e.executeToolCall(order, toolCall)

			e.mu.Lock()
			e.results = append(e.results, result)
			e.mu.Unlock()
		}()
	}
}

func ensureToolCallID(toolCall schema.ToolCall) schema.ToolCall {
	if strings.TrimSpace(toolCall.ID) != "" {
		return toolCall
	}

	toolName := strings.TrimSpace(toolCall.Function.Name)
	if toolName == "" {
		toolName = "tool"
	}

	fingerprint := md5.Sum([]byte(toolName + "\n" + strings.TrimSpace(toolCall.Function.Arguments)))
	toolCall.ID = fmt.Sprintf("auto_call_%s_%s", toolName, hex.EncodeToString(fingerprint[:8]))
	return toolCall
}

func normalizeToolCallIDs(toolCalls []schema.ToolCall) []schema.ToolCall {
	if len(toolCalls) == 0 {
		return toolCalls
	}

	normalized := make([]schema.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		normalized[i] = ensureToolCallID(tc)
	}
	return normalized
}

func (e *toolCallExecutor) Wait() []toolCallExecutionResult {
	e.wg.Wait()

	e.mu.Lock()
	defer e.mu.Unlock()

	results := append([]toolCallExecutionResult(nil), e.results...)
	sort.Slice(results, func(i, j int) bool {
		return results[i].order < results[j].order
	})
	return results
}

func (e *toolCallExecutor) WaitMedia() {
	e.mediaWg.Wait()
}

func (e *toolCallExecutor) executeToolCall(order int, toolCall schema.ToolCall) toolCallExecutionResult {
	resultMessage := &schema.Message{
		Role:       schema.Tool,
		ToolCallID: toolCall.ID,
	}

	state := e.manager.clientState
	toolName := toolCall.Function.Name
	transportType := ""
	if e.manager != nil && e.manager.serverTransport != nil {
		transportType = e.manager.serverTransport.GetTransportType()
	}
	toolObj, ok := mcp.GetToolByNameWithTransport(state.DeviceID, state.AgentID, transportType, toolName, state.DeviceConfig.MCPServiceNames)
	if !ok || toolObj == nil {
		log.Errorf("Tool not found: %s", toolName)
		resultMessage.Content = fmt.Sprintf("Tool not found: %s", toolName)
		return toolCallExecutionResult{order: order, message: resultMessage}
	}

	log.Infof("Make tool call request: %s, parameters: %+v", toolName, toolCall.Function.Arguments)
	startTs := time.Now().UnixMilli()
	fcResult, err := toolObj.InvokableRun(e.toolCtx, toolCall.Function.Arguments)
	if err != nil {
		log.Errorf("Tool call failed: %v", err)
		resultMessage.Content = fmt.Sprintf("Tool %s call failed: %v", toolName, err)
		return toolCallExecutionResult{order: order, message: resultMessage}
	}
	costTs := time.Now().UnixMilli() - startTs
	if len(fcResult) > 2048 {
		log.Infof("Tool call result len: %d, time consumption: %dms", len(fcResult), costTs)
	} else {
		log.Infof("Tool call result %s, time taken: %dms", fcResult, costTs)
	}

	execResult := toolCallExecutionResult{
		order:             order,
		message:           resultMessage,
		invokeToolSuccess: true,
	}
	normalizedContent := fcResult

	var contentList []mcp_go.Content
	if mcpResp, ok := e.manager.handleLocalToolResult(fcResult); ok {
		if mcpResp.GetType() == MCPResponseTypeAction && mcpResp.GetAction() == "exit_conversation" {
			execResult.findExitTool = true
		}
		if actionResp, ok := mcpResp.(*MCPActionResponse); ok {
			if actionResp.FinalAction || actionResp.NoFurtherResponse || actionResp.SilenceLLM {
				execResult.shouldStopLLMProcessing = true
			}
		}
		contentList = mcpResp.GetContent()
	} else if toolCallResult, ok := e.manager.handleToolResult(fcResult); ok {
		if toolCallResult.IsError {
			log.Errorf("Tool call failed: %s, error tag: %t", fcResult, toolCallResult.IsError)
		}
		contentList = toolCallResult.Content
	}

	if len(contentList) > 0 {
		var mcpContent string
		for _, content := range contentList {
			if audioContent, ok := content.(mcp_go.AudioContent); ok {
				log.Debugf("Calling tool %s returns audio resource length: %d", toolName, len(audioContent.Data))
				mcpContent = "Success"
				if err := e.manager.handleAudioContent(e.ctx, toolName, audioContent, &e.mediaWg); err != nil {
					log.Errorf("mcp failed to play audio resource: %v", err)
					mcpContent = "Failed"
				}
				execResult.shouldStopLLMProcessing = true
				execResult.hasMediaOutput = true
				break
			}
			if resourceLink, ok := content.(mcp_go.ResourceLink); ok {
				log.Debugf("Call tool %s to return resource link: %+v", toolName, resourceLink)
				mcpContent = "Success"
				if err := e.manager.handleResourceLink(e.ctx, resourceLink, toolObj, &e.mediaWg); err != nil {
					log.Errorf("mcp playback resource link failed: %v", err)
					mcpContent = "Failed"
				}
				execResult.shouldStopLLMProcessing = true
				execResult.hasMediaOutput = true
				break
			}
			if textContent, ok := content.(mcp_go.TextContent); ok {
				log.Debugf("Calling tool %s returns text resource length: %s", toolName, textContent.Text)
				mcpContent += textContent.Text
			}
		}
		if mcpContent != "" {
			normalizedContent = mcpContent
		}
	}

	execResult.message.Content = normalizedContent
	return execResult
}

func (l *LLMManager) handleResourceLink(ctx context.Context, resourceLink mcp_go.ResourceLink, toolCall tool.InvokableTool, wg *sync.WaitGroup) error {
	wg.Add(1)

	source, err := buildMediaSourceFromResourceLink(resourceLink, toolCall)
	if err != nil {
		wg.Done()
		return err
	}

	if l.session == nil || l.session.mediaPlayer == nil {
		wg.Done()
		return fmt.Errorf("session media player not initialized")
	}

	handle, err := l.session.mediaPlayer.PlaySourceWithHandle(ctx, source)
	if err != nil {
		wg.Done()
		return err
	}

	go func() {
		defer wg.Done()
		_ = handle.Wait(ctx)
	}()

	return nil
}

func (l *LLMManager) handleAudioContent(ctx context.Context, realMusicName string, audioContent mcp_go.AudioContent, wg *sync.WaitGroup) error {
	wg.Add(1)

	source, err := buildMediaSourceFromAudioContent(realMusicName, audioContent)
	if err != nil {
		wg.Done()
		log.Errorf("Failed to decode audio data: %v", err)
		return err
	}

	if l.session == nil || l.session.mediaPlayer == nil {
		wg.Done()
		return fmt.Errorf("session media player not initialized")
	}

	handle, err := l.session.mediaPlayer.PlaySourceWithHandle(ctx, source)
	if err != nil {
		wg.Done()
		return err
	}

	go func() {
		defer wg.Done()
		_ = handle.Wait(ctx)
	}()

	return nil
}

func buildMediaSourceFromAudioContent(title string, audioContent mcp_go.AudioContent) (MediaSourceDescriptor, error) {
	rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Data)
	if err != nil {
		return MediaSourceDescriptor{}, fmt.Errorf("Failed to decode audio data: %v", err)
	}

	title = strings.TrimSpace(title)
	if title == "" || title == "Success" {
		title = "tool audio"
	}

	return MediaSourceDescriptor{
		Title:      title,
		MIMEType:   audioContent.MIMEType,
		SourceType: MediaSourceTypeInlineAudio,
		Meta: map[string]string{
			"mime_type": audioContent.MIMEType,
			"source":    string(MediaSourceTypeInlineAudio),
		},
		Inline: &InlineAudioSource{
			Data: rawAudioData,
		},
	}, nil
}

func buildMediaSourceFromResourceLink(resourceLink mcp_go.ResourceLink, toolCall tool.InvokableTool) (MediaSourceDescriptor, error) {
	mcpTool, ok := toolCall.(*mcp.McpTool)
	if !ok || mcpTool == nil {
		return MediaSourceDescriptor{}, fmt.Errorf("resource link playback only supports MCP remote tools")
	}

	serverName := mcpTool.GetServerName()
	endpointSnapshot := mcp.GetServerEndpointSnapshotByName(serverName)
	directAudioURL := strings.TrimSpace(resourceLink.Description)
	readArgs := make(map[string]any)
	if directAudioURL != "" {
		readArgs["url"] = directAudioURL
	}

	title := strings.TrimSpace(resourceLink.Name)
	if title == "" {
		title = strings.TrimSpace(resourceLink.Description)
	}
	if title == "" {
		title = strings.TrimSpace(resourceLink.URI)
	}

	toolName := ""
	if info, err := mcpTool.Info(context.Background()); err == nil && info != nil {
		toolName = info.Name
	}

	return MediaSourceDescriptor{
		Title:      title,
		MIMEType:   resourceLink.MIMEType,
		SourceType: MediaSourceTypeMCPResource,
		Meta: map[string]string{
			"source":       string(MediaSourceTypeMCPResource),
			"server_name":  serverName,
			"endpoint":     endpointSnapshot,
			"resource_uri": resourceLink.URI,
			"audio_url":    directAudioURL,
			"tool_name":    toolName,
		},
		MCP: &MCPMediaSource{
			ServerName:       serverName,
			EndpointSnapshot: endpointSnapshot,
			ToolName:         toolName,
			ResourceURI:      resourceLink.URI,
			DirectAudioURL:   directAudioURL,
			Description:      resourceLink.Description,
			ReadArgs:         readArgs,
			PageSize:         McpReadResourcePageSize,
			Client:           mcpTool.GetClient(),
		},
	}, nil
}

func (l *LLMManager) handleLocalToolResult(toolResult string) (MCPResponse, bool) {
	// If it is mcp return format, parse
	var response MCPResponse
	var err error
	if response, err = ParseMCPResponse(toolResult); err != nil {
		return nil, false
	}
	return response, true
}

func (l *LLMManager) handleToolResult(toolResultStr string) (mcp_go.CallToolResult, bool) {
	var toolResult mcp_go.CallToolResult
	if err := json.Unmarshal([]byte(toolResultStr), &toolResult); err != nil {
		log.Debugf("Failed to unmarshal tool result: %v", err)
		toolResult.Content = []mcp_go.Content{
			mcp_go.TextContent{
				Type: "text",
				Text: toolResultStr,
			},
		}
		return toolResult, true
	}

	return toolResult, true
}
