package plugins

import "xiaozhi-esp32-server-golang/internal/domain/chat/streamtransform"

// Init initializes the output related transform.
func Init(registry *streamtransform.Registry) {
	if registry == nil {
		return
	}

	// Register output shaping plug-in (text segmentation + tool call closing)
	RegisterOutputSegmenter(registry)
}
