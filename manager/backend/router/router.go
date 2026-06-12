package router

import (
	"io/fs"
	"net/http"
	"xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/controllers"
	"xiaozhi/manager/backend/middleware"
	"xiaozhi/manager/backend/static"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	internalAuthToken := config.ResolveInternalAuthToken(cfg)
	endpointAuthToken := config.ResolveEndpointAuthToken(cfg)

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-Token"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Initialize controllers
	authController := &controllers.AuthController{DB: db}
	webSocketController := controllers.NewWebSocketController(db, endpointAuthToken)
	adminController := &controllers.AdminController{
		DB:                  db,
		WebSocketController: webSocketController,
		InternalAuthToken:   internalAuthToken,
		EndpointAuthToken:   endpointAuthToken,
		AppServerURL:        cfg.GetAppServerURL(),
	}
	userController := &controllers.UserController{
		DB:                  db,
		WebSocketController: webSocketController,
		InternalAuthToken:   internalAuthToken,
		EndpointAuthToken:   endpointAuthToken,
	}
	deviceActivationController := &controllers.DeviceActivationController{DB: db}
	setupController := &controllers.SetupController{DB: db}
	speakerGroupController := controllers.NewSpeakerGroupController(db, cfg)
	voiceCloneController := controllers.NewVoiceCloneController(db, cfg)
	poolStatsController := controllers.NewPoolStatsController()

	// Initialize chat history controller (use the provided cfg to avoid wrong path when embedded)
	audioBasePath := "./storage/chat_history/audio"
	maxFileSize := int64(10 * 1024 * 1024) // default 10MB
	if cfg.History.AudioBasePath != "" {
		audioBasePath = cfg.History.AudioBasePath
	}
	if cfg.History.MaxFileSize > 0 {
		maxFileSize = cfg.History.MaxFileSize
	}
	chatHistoryController := &controllers.ChatHistoryController{
		DB:            db,
		AudioBasePath: audioBasePath,
		MaxFileSize:   maxFileSize,
	}

	// API route group
	api := r.Group("/api")
	{
		// Public routes (no auth required)
		api.GET("/captcha/status", authController.GetCaptchaStatus)
		api.GET("/captcha/challenge", authController.GetSimpleCaptcha)
		api.POST("/login", authController.Login)
		api.POST("/register", authController.Register)

		// Database initialization routes (no auth required)
		api.GET("/setup/status", setupController.CheckSetupStatus)
		api.POST("/setup/initialize", setupController.InitializeDatabase)

		// Internal service routes (service-to-service token auth)
		internal := api.Group("")
		internal.Use(middleware.InternalServiceAuth(internalAuthToken))
		{
			internal.GET("/internal/device/check-activation", deviceActivationController.CheckDeviceActivation)
			internal.GET("/internal/device/activation-info", deviceActivationController.GetActivationInfo)
			internal.POST("/internal/device/activate", deviceActivationController.ActivateDevice)

			internal.GET("/configs", adminController.GetDeviceConfigs)
			internal.GET("/system/configs", adminController.GetSystemConfigs)
			internal.POST("/internal/history/messages", chatHistoryController.SaveMessage)                         // save message (internal service endpoint)
			internal.PUT("/internal/history/messages/:message_id/audio", chatHistoryController.UpdateMessageAudio) // update message audio (internal service endpoint)
			internal.GET("/internal/history/messages", chatHistoryController.GetMessagesForInit)                   // get messages for init load (internal service endpoint)
			internal.POST("/internal/pool/stats", poolStatsController.ReportPoolStats)                             // report pool stats (internal service endpoint)
			internal.POST("/internal/devices/:device_name/switch-role", adminController.SwitchDeviceRoleByNameInternal)
			internal.POST("/internal/devices/:device_name/restore-default-role", adminController.RestoreDeviceDefaultRoleInternal)
		}

		// Authenticated routes
		auth := api.Group("")
		auth.Use(middleware.JWTAuth())
		{
			auth.GET("/profile", authController.GetProfile)
			// General endpoint for device info in the system
			auth.GET("/dashboard/stats", userController.GetDashboardStats)
			// Device role endpoint (accessible by admin and regular users; controller enforces permissions)
			auth.POST("/devices/:id/apply-role", adminController.ApplyRoleToDevice)

			// Role management (primary path)
			auth.GET("/roles", adminController.GetRolesNew)
			auth.GET("/roles/:id", adminController.GetRoleNew)
			auth.POST("/roles", adminController.CreateRoleNew)
			auth.PUT("/roles/:id", adminController.UpdateRoleNew)
			auth.DELETE("/roles/:id", adminController.DeleteRoleNew)
			auth.PATCH("/roles/:id/toggle", adminController.ToggleRoleStatus)

			// User routes
			user := auth.Group("/user")
			{
				// Role management
				user.GET("/roles", adminController.GetRolesNew)
				user.GET("/roles/:id", adminController.GetRoleNew)
				user.POST("/roles", adminController.CreateRoleNew)
				user.PUT("/roles/:id", adminController.UpdateRoleNew)
				user.DELETE("/roles/:id", adminController.DeleteRoleNew)
				user.PATCH("/roles/:id/toggle", adminController.ToggleRoleStatus)

				// API tokens (for OpenAPI calls)
				user.GET("/api-tokens", userController.ListAPITokens)
				user.POST("/api-tokens", userController.CreateAPIToken)
				user.DELETE("/api-tokens/:id", userController.RevokeAPIToken)

				// Device management
				user.GET("/devices", userController.GetMyDevices)
				user.POST("/devices", userController.CreateDevice)
				user.PUT("/devices/:id", userController.UpdateDevice)
				user.DELETE("/devices/:id", userController.DeleteDevice)

				// Agent management
				user.GET("/agents", userController.GetAgents)
				user.POST("/agents", userController.CreateAgent)
				user.GET("/agents/:id", userController.GetAgent)
				user.PUT("/agents/:id", userController.UpdateAgent)
				user.DELETE("/agents/:id", userController.DeleteAgent)
				user.GET("/agents/:id/devices", userController.GetAgentDevices)
				user.POST("/agents/:id/devices", userController.AddDeviceToAgent)
				user.DELETE("/agents/:id/devices/:device_id", userController.RemoveDeviceFromAgent)
				user.GET("/agents/:id/knowledge-bases", userController.GetAgentKnowledgeBases)
				user.PUT("/agents/:id/knowledge-bases", userController.UpdateAgentKnowledgeBases)

				// User knowledge base management (plain text)
				user.GET("/knowledge-bases", userController.GetKnowledgeBases)
				user.POST("/knowledge-bases", userController.CreateKnowledgeBase)
				user.GET("/knowledge-bases/:id", userController.GetKnowledgeBase)
				user.PUT("/knowledge-bases/:id", userController.UpdateKnowledgeBase)
				user.DELETE("/knowledge-bases/:id", userController.DeleteKnowledgeBase)
				user.POST("/knowledge-bases/:id/sync", userController.SyncKnowledgeBase)
				user.POST("/knowledge-bases/:id/test-search", userController.TestKnowledgeBaseSearch)
				user.GET("/knowledge-bases/:id/documents", userController.GetKnowledgeBaseDocuments)
				user.POST("/knowledge-bases/:id/documents", userController.CreateKnowledgeBaseDocument)
				user.POST("/knowledge-bases/:id/documents/upload", userController.CreateKnowledgeBaseDocumentByUpload)
				user.PUT("/knowledge-bases/:id/documents/:doc_id", userController.UpdateKnowledgeBaseDocument)
				user.DELETE("/knowledge-bases/:id/documents/:doc_id", userController.DeleteKnowledgeBaseDocument)
				user.POST("/knowledge-bases/:id/documents/:doc_id/sync", userController.SyncKnowledgeBaseDocument)

				// Role templates and voice options
				user.GET("/role-templates", userController.GetRoleTemplates)
				user.GET("/voice-options", userController.GetVoiceOptions)
				user.GET("/voice-clone/capabilities", voiceCloneController.GetCloneProviderCapabilities)
				user.POST("/voice-clones", voiceCloneController.CreateVoiceClone)
				user.GET("/voice-clones", voiceCloneController.GetVoiceClones)
				user.PUT("/voice-clones/:id", voiceCloneController.UpdateVoiceClone)
				user.DELETE("/voice-clones/:id", voiceCloneController.DeleteVoiceClone)
				user.POST("/voice-clones/:id/retry", voiceCloneController.RetryVoiceClone)
				user.POST("/voice-clones/:id/append-audio", voiceCloneController.AppendVoiceCloneAudio)
				user.GET("/voice-clones/:id/preview", voiceCloneController.PreviewClonedVoice)
				user.GET("/voice-clones/:id/audios", voiceCloneController.GetVoiceCloneAudios)
				user.GET("/voice-clones/audios/:audio_id/file", voiceCloneController.GetVoiceCloneAudioFile)

				// Config lists
				user.GET("/llm-configs", userController.GetLLMConfigs)
				user.GET("/llm-configs/options", userController.GetLLMConfigs)
				user.GET("/tts-configs", userController.GetTTSConfigs)
				user.GET("/tts-configs/options", userController.GetTTSConfigs)

				// MCP endpoints
				user.GET("/mcp-services/options", userController.GetMCPServiceOptions)
				user.GET("/agents/:id/mcp-services/options", userController.GetAgentMCPServiceOptions)
				user.GET("/agents/:id/mcp-endpoint", userController.GetAgentMCPEndpoint)
				user.GET("/agents/:id/openclaw-endpoint", userController.GetAgentOpenClawEndpoint)
				user.POST("/agents/:id/openclaw-chat-test", userController.CallAgentOpenClawChatTest)
				user.GET("/agents/:id/mcp-tools", userController.GetAgentMcpTools)
				user.POST("/agents/:id/mcp-call", userController.CallAgentMcpTool)
				user.GET("/devices/:id/mcp-tools", userController.GetDeviceMcpTools)
				user.POST("/devices/:id/mcp-call", userController.CallDeviceMcpTool)

				// Voice push
				user.POST("/devices/inject-message", userController.InjectMessage)

				// Speaker group management
				user.POST("/speaker-groups", speakerGroupController.CreateSpeakerGroup)
				user.GET("/speaker-groups", speakerGroupController.GetSpeakerGroups)
				user.GET("/speaker-groups/:id", speakerGroupController.GetSpeakerGroup)
				user.PUT("/speaker-groups/:id", speakerGroupController.UpdateSpeakerGroup)
				user.DELETE("/speaker-groups/:id", speakerGroupController.DeleteSpeakerGroup)
				user.POST("/speaker-groups/:id/verify", speakerGroupController.VerifySpeakerGroup)

				// Speaker sample management (note: uses :id not :group_id to avoid route conflicts)
				user.POST("/speaker-groups/:id/samples", speakerGroupController.AddSample)
				user.GET("/speaker-groups/:id/samples", speakerGroupController.GetSamples)
				user.GET("/speaker-groups/:id/samples/:sample_id/file", speakerGroupController.GetSampleFile)
				user.DELETE("/speaker-groups/:id/samples/:sample_id", speakerGroupController.DeleteSample)

				// Chat history
				user.GET("/history/messages", chatHistoryController.GetMessages)
				user.DELETE("/history/messages/:id", chatHistoryController.DeleteMessage)
				user.GET("/history/export", chatHistoryController.ExportMessages)
				user.GET("/history/agents/:agent_id/messages", chatHistoryController.GetMessagesByAgent)
				user.GET("/history/messages/:id/audio", chatHistoryController.GetAudioFile)
			}

			// External OpenAPI routes (supports JWT or API token)
			openV1 := api.Group("/open/v1")
			openV1.Use(middleware.OpenAPIAuth(db))
			{
				openV1.GET("/profile", authController.GetProfile)
				openV1.GET("/devices", userController.GetMyDevices)
				openV1.POST("/devices", userController.CreateDevice)
				openV1.GET("/agents", userController.GetAgents)
				openV1.POST("/agents", userController.CreateAgent)
				openV1.GET("/agents/:id", userController.GetAgent)
				openV1.PUT("/agents/:id", userController.UpdateAgent)
				openV1.DELETE("/agents/:id", userController.DeleteAgent)
				openV1.GET("/history/messages", chatHistoryController.GetMessages)
				openV1.GET("/history/export", chatHistoryController.ExportMessages)
				openV1.POST("/devices/inject-message", userController.InjectMessage)
				openV1.GET("/agents/:id/mcp-tools", userController.GetAgentMcpTools)
				openV1.POST("/agents/:id/mcp-call", userController.CallAgentMcpTool)
				openV1.GET("/devices/:id/mcp-tools", userController.GetDeviceMcpTools)
				openV1.POST("/devices/:id/mcp-call", userController.CallDeviceMcpTool)
			}

			// Admin routes
			admin := auth.Group("/admin")
			admin.Use(middleware.AdminAuth())
			{
				// General config management
				admin.GET("/configs", adminController.GetConfigs)
				admin.POST("/configs", adminController.CreateConfig)
				admin.GET("/configs/:id", adminController.GetConfig)
				admin.PUT("/configs/:id", adminController.UpdateConfig)
				admin.DELETE("/configs/:id", adminController.DeleteConfig)
				admin.POST("/configs/:id/toggle", adminController.ToggleConfigEnable)

				// Specific config type routes (frontend compatibility)
				admin.GET("/vad-configs", adminController.GetVADConfigs)
				admin.POST("/vad-configs", adminController.CreateVADConfig)
				admin.PUT("/vad-configs/:id", adminController.UpdateVADConfig)
				admin.DELETE("/vad-configs/:id", adminController.DeleteVADConfig)

				admin.GET("/asr-configs", adminController.GetASRConfigs)
				admin.POST("/asr-configs", adminController.CreateASRConfig)
				admin.PUT("/asr-configs/:id", adminController.UpdateASRConfig)
				admin.DELETE("/asr-configs/:id", adminController.DeleteASRConfig)

				admin.GET("/llm-configs", adminController.GetLLMConfigs)
				admin.POST("/llm-configs", adminController.CreateLLMConfig)
				admin.PUT("/llm-configs/:id", adminController.UpdateLLMConfig)
				admin.DELETE("/llm-configs/:id", adminController.DeleteLLMConfig)
				admin.POST("/llm-configs/test-connection", adminController.TestLLMConnection)
				admin.POST("/llm-configs/fetch-models", adminController.FetchLLMModels)

				admin.GET("/tts-configs", adminController.GetTTSConfigs)
				admin.POST("/tts-configs", adminController.CreateTTSConfig)
				admin.PUT("/tts-configs/:id", adminController.UpdateTTSConfig)
				admin.DELETE("/tts-configs/:id", adminController.DeleteTTSConfig)
				admin.GET("/supertonic-model", adminController.GetSupertonicModelStatus)
				admin.POST("/supertonic-model/download", adminController.DownloadSupertonicModel)
				admin.POST("/tts-preview", adminController.PreviewTTSAudio)

				admin.GET("/speaker-configs", adminController.GetSpeakerConfigs)
				admin.POST("/speaker-configs", adminController.CreateSpeakerConfig)
				admin.PUT("/speaker-configs/:id", adminController.UpdateSpeakerConfig)
				admin.DELETE("/speaker-configs/:id", adminController.DeleteSpeakerConfig)

				admin.GET("/vision-configs", adminController.GetVisionConfigs)
				admin.POST("/vision-configs", adminController.CreateVisionConfig)
				admin.PUT("/vision-configs/:id", adminController.UpdateVisionConfig)
				admin.DELETE("/vision-configs/:id", adminController.DeleteVisionConfig)

				// Vision base config
				admin.GET("/vision-base-config", adminController.GetVisionBaseConfig)
				admin.PUT("/vision-base-config", adminController.UpdateVisionBaseConfig)

				// Chat settings
				admin.GET("/chat-settings", adminController.GetChatSettings)
				admin.PUT("/chat-settings", adminController.UpdateChatSettings)

				admin.GET("/ota-configs", adminController.GetOTAConfigs)
				admin.POST("/ota-configs", adminController.CreateOTAConfig)
				admin.PUT("/ota-configs/:id", adminController.UpdateOTAConfig)
				admin.DELETE("/ota-configs/:id", adminController.DeleteOTAConfig)

				admin.GET("/mqtt-configs", adminController.GetMQTTConfigs)
				admin.POST("/mqtt-configs", adminController.CreateMQTTConfig)
				admin.PUT("/mqtt-configs/:id", adminController.UpdateMQTTConfig)
				admin.DELETE("/mqtt-configs/:id", adminController.DeleteMQTTConfig)

				admin.GET("/mqtt-server-configs", adminController.GetMQTTServerConfigs)
				admin.POST("/mqtt-server-configs", adminController.CreateMQTTServerConfig)
				admin.PUT("/mqtt-server-configs/:id", adminController.UpdateMQTTServerConfig)
				admin.DELETE("/mqtt-server-configs/:id", adminController.DeleteMQTTServerConfig)

				admin.GET("/mqtt-server-status", adminController.GetMqttServerStatus)
				admin.GET("/mqtt-status", adminController.GetMqttClientStatus)

				admin.GET("/udp-configs", adminController.GetUDPConfigs)
				admin.POST("/udp-configs", adminController.CreateUDPConfig)
				admin.PUT("/udp-configs/:id", adminController.UpdateUDPConfig)
				admin.DELETE("/udp-configs/:id", adminController.DeleteUDPConfig)

				admin.GET("/mcp-configs", adminController.GetMCPConfigs)
				admin.POST("/mcp-configs", adminController.CreateMCPConfig)
				admin.POST("/mcp-configs/discover-tools", adminController.DiscoverMCPConfigTools)
				admin.PUT("/mcp-configs/:id", adminController.UpdateMCPConfig)
				admin.DELETE("/mcp-configs/:id", adminController.DeleteMCPConfig)
				admin.GET("/mcp-markets", adminController.GetMCPMarkets)
				admin.POST("/mcp-markets", adminController.CreateMCPMarket)
				admin.PUT("/mcp-markets/:id", adminController.UpdateMCPMarket)
				admin.DELETE("/mcp-markets/:id", adminController.DeleteMCPMarket)
				admin.POST("/mcp-markets/:id/test", adminController.TestMCPMarket)
				admin.GET("/mcp-market/providers", adminController.GetMCPMarketProviders)
				admin.GET("/mcp-market/services", adminController.GetMCPMarketServices)
				admin.GET("/mcp-market/services/:market_id/*service_id", adminController.GetMCPMarketServiceDetail)
				admin.POST("/mcp-market/import", adminController.ImportMCPMarketService)
				admin.GET("/mcp-market/imported-services", adminController.GetMCPMarketImportedServices)
				admin.POST("/mcp-market/imported-services", adminController.CreateMCPMarketImportedService)
				admin.GET("/mcp-market/imported-services/:id/tools", adminController.GetMCPMarketImportedServiceTools)
				admin.PUT("/mcp-market/imported-services/:id", adminController.UpdateMCPMarketImportedService)
				admin.DELETE("/mcp-market/imported-services/:id", adminController.DeleteMCPMarketImportedService)

				// Memory config management
				admin.GET("/memory-configs", adminController.GetMemoryConfigs)
				admin.POST("/memory-configs", adminController.CreateMemoryConfig)
				admin.PUT("/memory-configs/:id", adminController.UpdateMemoryConfig)
				admin.DELETE("/memory-configs/:id", adminController.DeleteMemoryConfig)
				admin.POST("/memory-configs/:id/set-default", adminController.SetDefaultMemoryConfig)

				// Knowledge search config management (provider API calls)
				admin.GET("/knowledge-search-configs", adminController.GetKnowledgeSearchConfigs)
				admin.POST("/knowledge-search-configs", adminController.CreateKnowledgeSearchConfig)
				admin.PUT("/knowledge-search-configs/:id", adminController.UpdateKnowledgeSearchConfig)
				admin.DELETE("/knowledge-search-configs/:id", adminController.DeleteKnowledgeSearchConfig)
				admin.POST("/knowledge-search-configs/weknora/models", adminController.ListWeknoraModels)

				// Global role management (kept for legacy API compatibility)
				admin.GET("/global-roles", adminController.GetGlobalRoles)
				admin.POST("/global-roles", adminController.CreateGlobalRole)
				admin.PUT("/global-roles/:id", adminController.UpdateGlobalRole)
				admin.DELETE("/global-roles/:id", adminController.DeleteGlobalRole)

				// Global role management (new API)
				admin.GET("/roles", adminController.GetRolesNew)
				admin.GET("/roles/global", adminController.GetGlobalRolesNew)
				admin.POST("/roles/global", adminController.CreateRoleNew)
				admin.PUT("/roles/global/:id", adminController.UpdateRoleNew)
				admin.DELETE("/roles/global/:id", adminController.DeleteRoleNew)
				admin.PATCH("/roles/global/:id/toggle", adminController.ToggleRoleStatus)
				admin.PATCH("/roles/global/:id/default", adminController.SetDefaultRole)

				// Device management
				admin.GET("/devices", adminController.GetDevices)
				admin.GET("/devices/validate-code", adminController.ValidateDeviceCode)
				admin.POST("/devices", adminController.CreateDevice)
				admin.PUT("/devices/:id", adminController.UpdateDevice)
				admin.DELETE("/devices/:id", adminController.DeleteDevice)

				// Agent management
				admin.GET("/agents", adminController.GetAgents)
				admin.POST("/agents", adminController.CreateAgent)
				admin.PUT("/agents/:id", adminController.UpdateAgent)
				admin.DELETE("/agents/:id", adminController.DeleteAgent)
				admin.GET("/agents/:id/mcp-endpoint", adminController.GetAgentMCPEndpoint)
				admin.GET("/agents/:id/openclaw-endpoint", adminController.GetAgentOpenClawEndpoint)
				admin.POST("/agents/:id/openclaw-chat-test", adminController.CallAgentOpenClawChatTest)
				admin.GET("/agents/:id/mcp-tools", adminController.GetAgentMcpTools)
				admin.POST("/agents/:id/mcp-call", adminController.CallAgentMcpTool)
				admin.GET("/devices/:id/mcp-tools", adminController.GetDeviceMcpTools)
				admin.POST("/devices/:id/mcp-call", adminController.CallDeviceMcpTool)

				// User management
				admin.GET("/users/options", adminController.GetUserOptions)
				admin.GET("/users", adminController.GetUsers)
				admin.POST("/users", adminController.CreateUser)
				admin.PUT("/users/:id", adminController.UpdateUser)
				admin.DELETE("/users/:id", adminController.DeleteUser)
				admin.POST("/users/:id/reset-password", adminController.ResetUserPassword)

				admin.GET("/users/:id/knowledge-bases", adminController.GetUserKnowledgeBasesAdmin)
				admin.POST("/users/:id/knowledge-bases", adminController.CreateUserKnowledgeBaseAdmin)
				admin.PUT("/users/:id/knowledge-bases/:kb_id", adminController.UpdateUserKnowledgeBaseAdmin)
				admin.DELETE("/users/:id/knowledge-bases/:kb_id", adminController.DeleteUserKnowledgeBaseAdmin)

				admin.GET("/users/:id/voice-options", adminController.GetUserVoiceOptionsAdmin)
				admin.GET("/users/:id/voice-clones", adminController.GetUserVoiceClonesAdmin)
				admin.GET("/users/:id/voice-clone-quotas", adminController.GetUserVoiceCloneQuotas)
				admin.PUT("/users/:id/voice-clone-quotas", adminController.UpdateUserVoiceCloneQuotas)

				// Config import/export
				admin.GET("/configs/export", adminController.ExportConfigs)
				admin.POST("/configs/import", adminController.ImportConfigs)
				// One-click config test (OTA runs inside manager; VAD/ASR/LLM/TTS forwarded via WebSocket to main service)
				admin.POST("/configs/test", adminController.TestConfigs)

				// Connection pool statistics
				admin.GET("/pool/stats", poolStatsController.GetPoolStats)
				admin.GET("/pool/stats/summary", poolStatsController.GetPoolStatsSummary)
			}
		}
	}

	// WebSocket route
	r.GET("/ws", webSocketController.HandleWebSocket)

	// Embedded frontend static assets (built with -tags embed_ui):
	// on NoRoute, try static file first, then SPA fallback
	if sub, err := fs.Sub(static.FS, "dist"); err == nil {
		r.NoRoute(serveEmbedStatic(sub))
	}

	return r
}

// serveEmbedStatic serves embedded static files on unmatched routes;
// returns the matching file for GET requests, or falls back to index.html for SPA routing.
func serveEmbedStatic(fsys fs.FS) gin.HandlerFunc {
	indexHTML, _ := fs.ReadFile(fsys, "index.html")
	fileServer := http.FileServer(http.FS(fsys))
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		path := c.Request.URL.Path
		if path == "" || path[0] != '/' {
			path = "/" + path
		}
		if path == "/" {
			path = "/index.html"
		}
		name := path[1:]
		if _, err := fs.Stat(fsys, name); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		if len(indexHTML) > 0 {
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
			return
		}
		c.Status(http.StatusNotFound)
	}
}
