package controllers

import (
	"strings"
	"testing"

	"xiaozhi/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAgentDeviceServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Config{},
		&models.Agent{},
		&models.Device{},
		&models.KnowledgeBase{},
		&models.AgentKnowledgeBase{},
		&models.Role{},
		&models.MCPMarketService{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func createServiceTestUser(t *testing.T, db *gorm.DB, username, role string) models.User {
	t.Helper()
	user := models.User{
		Username: username,
		Email:    username + "@example.test",
		Password: "secret",
		Role:     role,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return user
}

func createServiceTestConfig(t *testing.T, db *gorm.DB, typ, id, provider string) {
	t.Helper()
	if err := db.Create(&models.Config{
		Type:      typ,
		ConfigID:  id,
		Name:      id,
		Provider:  provider,
		Enabled:   true,
		IsDefault: true,
	}).Error; err != nil {
		t.Fatalf("create config %s/%s: %v", typ, id, err)
	}
}

func createServiceTestKnowledgeBase(t *testing.T, db *gorm.DB, userID uint, name string) models.KnowledgeBase {
	t.Helper()
	kb := models.KnowledgeBase{UserID: userID, Name: name, Content: "content", Status: "active"}
	if err := db.Create(&kb).Error; err != nil {
		t.Fatalf("create knowledge base %s: %v", name, err)
	}
	return kb
}

func strPtr(value string) *string {
	return &value
}

func TestAgentServicePermissionVoiceAndKnowledgeLinks(t *testing.T) {
	db := setupAgentDeviceServiceTestDB(t)
	userA := createServiceTestUser(t, db, "user-a", "user")
	userB := createServiceTestUser(t, db, "user-b", "user")
	createServiceTestConfig(t, db, "llm", "llm-default", "openai")
	createServiceTestConfig(t, db, "tts", "tts-default", "doubao")
	kbA := createServiceTestKnowledgeBase(t, db, userA.ID, "kb-a")
	kbB := createServiceTestKnowledgeBase(t, db, userB.ID, "kb-b")

	agentSvc := NewAgentService(db)
	kbIDs := []uint{kbA.ID}
	agent, err := agentSvc.Create(accessScope{ActorUserID: userA.ID}, AgentPayload{
		UserID:           userB.ID,
		Name:             "agent-a",
		Nickname:         strPtr("assistant-a"),
		CustomPrompt:     "prompt",
		LLMConfigID:      strPtr("llm-default"),
		TTSConfigID:      strPtr("tts-default"),
		Voice:            strPtr("exact-voice"),
		KnowledgeBaseIDs: &kbIDs,
	})
	if err != nil {
		t.Fatalf("create user agent: %v", err)
	}
	if agent.UserID != userA.ID {
		t.Fatalf("agent user_id = %d, want %d", agent.UserID, userA.ID)
	}
	if agent.Voice == nil || *agent.Voice != "exact-voice" {
		t.Fatalf("agent voice = %#v, want exact-voice", agent.Voice)
	}
	if len(agent.KnowledgeBaseIDs) != 1 || agent.KnowledgeBaseIDs[0] != kbA.ID {
		t.Fatalf("knowledge links = %#v, want [%d]", agent.KnowledgeBaseIDs, kbA.ID)
	}

	crossUserKBIDs := []uint{kbB.ID}
	if _, err := agentSvc.Update(accessScope{ActorUserID: userA.ID}, agent.ID, AgentPayload{
		Name:             "agent-a",
		Nickname:         strPtr("assistant-a"),
		KnowledgeBaseIDs: &crossUserKBIDs,
	}); err == nil || !strings.Contains(err.Error(), "knowledge base") {
		t.Fatalf("cross-user knowledge update error = %v, want knowledge ownership rejection", err)
	}

	if _, err := agentSvc.Get(accessScope{ActorUserID: userB.ID}, agent.ID); err == nil {
		t.Fatalf("other normal user should not read agent")
	}
	if _, err := agentSvc.Get(accessScope{ActorUserID: userB.ID, IsAdmin: true}, agent.ID); err != nil {
		t.Fatalf("admin should read agent: %v", err)
	}

	if _, err := agentSvc.Create(accessScope{ActorUserID: userB.ID, IsAdmin: true}, AgentPayload{
		UserID:           userA.ID,
		Name:             "admin-cross-kb",
		Nickname:         strPtr("admin-cross-kb"),
		KnowledgeBaseIDs: &crossUserKBIDs,
	}); err == nil || !strings.Contains(err.Error(), "knowledge base") {
		t.Fatalf("admin cross-user knowledge create error = %v, want rejection", err)
	}
}

func TestDeviceServiceBindingEnrichmentAndCrossUserRejection(t *testing.T) {
	db := setupAgentDeviceServiceTestDB(t)
	userA := createServiceTestUser(t, db, "device-user-a", "user")
	userB := createServiceTestUser(t, db, "device-user-b", "user")

	agentA := models.Agent{UserID: userA.ID, Name: "agent-a", Nickname: "agent-a"}
	agentB := models.Agent{UserID: userB.ID, Name: "agent-b", Nickname: "agent-b"}
	if err := db.Create(&agentA).Error; err != nil {
		t.Fatalf("create agent a: %v", err)
	}
	if err := db.Create(&agentB).Error; err != nil {
		t.Fatalf("create agent b: %v", err)
	}

	unbound := models.Device{DeviceCode: "123456", DeviceName: "dev-a", NickName: "dev-a"}
	if err := db.Create(&unbound).Error; err != nil {
		t.Fatalf("create unbound device: %v", err)
	}
	ownedByB := models.Device{UserID: userB.ID, AgentID: agentB.ID, DeviceCode: "654321", DeviceName: "dev-b", NickName: "dev-b", Activated: true}
	if err := db.Create(&ownedByB).Error; err != nil {
		t.Fatalf("create owned device: %v", err)
	}

	deviceSvc := NewDeviceService(db)
	bound, err := deviceSvc.BindToAgent(accessScope{ActorUserID: userA.ID}, agentA.ID, DevicePayload{
		Code:     "123456",
		NickName: "living room",
	})
	if err != nil {
		t.Fatalf("bind unbound device: %v", err)
	}
	if bound.UserID != userA.ID || bound.AgentID != agentA.ID || !bound.Activated {
		t.Fatalf("bound device = user:%d agent:%d activated:%v, want user:%d agent:%d active", bound.UserID, bound.AgentID, bound.Activated, userA.ID, agentA.ID)
	}
	if bound.AgentName != "agent-a" {
		t.Fatalf("bound agent name = %q, want agent-a", bound.AgentName)
	}

	if _, err := deviceSvc.BindToAgent(accessScope{ActorUserID: userA.ID}, agentA.ID, DevicePayload{Code: "654321"}); err == nil {
		t.Fatalf("binding device owned by another user should fail")
	}

	if _, err := deviceSvc.Update(accessScope{ActorUserID: userA.ID, IsAdmin: true}, bound.ID, DevicePayload{
		UserID:   userB.ID,
		NickName: "cross",
		AgentID:  agentA.ID,
	}); err == nil || !strings.Contains(err.Error(), "agent") {
		t.Fatalf("admin cross-user device-agent update error = %v, want rejection", err)
	}

	updated, err := deviceSvc.Update(accessScope{ActorUserID: userA.ID, IsAdmin: true}, bound.ID, DevicePayload{
		UserID:     userB.ID,
		NickName:   "moved",
		DeviceCode: "123456",
		DeviceName: "dev-a",
		AgentID:    agentB.ID,
	})
	if err != nil {
		t.Fatalf("admin same-user device-agent update: %v", err)
	}
	if updated.UserID != userB.ID || updated.AgentID != agentB.ID || updated.AgentName != "agent-b" || updated.Username != userB.Username {
		t.Fatalf("updated device enrichment = user:%d agent:%d agentName:%q username:%q", updated.UserID, updated.AgentID, updated.AgentName, updated.Username)
	}
}
