package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// HuggingFace repo structure after git clone:
//   {clone_root}/onnx/          ← tts.json + *.onnx + unicode_indexer.json
//   {clone_root}/voice_styles/  ← M1.json, F1.json, …
// So onnx_dir = {clone_root}/onnx

const supertonicHuggingFaceRepo = "https://huggingface.co/Supertone/supertonic-3"

func defaultSupertonicOnnxDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/home/user"
	}
	return filepath.Join(home, ".cache", "supertonic-model", "onnx")
}

func supertonicExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// GetSupertonicModelStatus checks whether the Supertonic ONNX model directory exists.
// path param = onnx_dir value (e.g. ~/.cache/supertonic-model/onnx)
// GET /admin/supertonic-model?path=~/.cache/supertonic-model/onnx
func (ac *AdminController) GetSupertonicModelStatus(c *gin.Context) {
	onnxDir := supertonicExpandHome(c.Query("path"))
	defPath := defaultSupertonicOnnxDir()
	if onnxDir == "" {
		onnxDir = defPath
	}

	_, err := os.Stat(filepath.Join(onnxDir, "tts.json"))
	c.JSON(http.StatusOK, gin.H{
		"exists":       err == nil,
		"onnx_dir":     onnxDir,
		"default_path": defPath,
	})
}

// DownloadSupertonicModel clones the Supertonic model from HuggingFace via git + git-lfs.
// Accepts onnx_dir; derives clone root as parent(onnx_dir).
// POST /admin/supertonic-model/download
// Body: {"onnx_dir": "/opt/supertonic-model/onnx"} (optional)
func (ac *AdminController) DownloadSupertonicModel(c *gin.Context) {
	var req struct {
		OnnxDir string `json:"onnx_dir"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.OnnxDir == "" {
		req.OnnxDir = defaultSupertonicOnnxDir()
	}
	req.OnnxDir = supertonicExpandHome(req.OnnxDir)
	cloneRoot := filepath.Dir(req.OnnxDir)

	// Already exists — return immediately
	if _, err := os.Stat(filepath.Join(req.OnnxDir, "tts.json")); err == nil {
		c.JSON(http.StatusOK, gin.H{"ok": true, "onnx_dir": req.OnnxDir, "already_exists": true})
		return
	}

	if err := os.MkdirAll(cloneRoot, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("cannot create directory: %v", err)})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Ensure git-lfs is initialised (required for large binary model files)
	if out, err := exec.CommandContext(ctx, "git", "lfs", "install").CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("git-lfs not available (run: sudo apt-get install git-lfs). Detail: %s", string(out)),
		})
		return
	}

	// git clone into cloneRoot — git-lfs downloads binary files automatically
	if out, err := exec.CommandContext(ctx, "git", "clone", supertonicHuggingFaceRepo, cloneRoot).CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("git clone failed: %s", strings.TrimSpace(string(out))),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "onnx_dir": req.OnnxDir})
}
