package handler_chat

import (
	"testing"

	"github.com/ixayda/ix-Go/internal/pkg/config"
)

func TestNormalizeAttachmentPresignRequestVideoMetadata(t *testing.T) {
	tests := []struct {
		name            string
		request         attachmentPresignRequest
		wantFileName    string
		wantContentType string
		wantIsVideo     bool
	}{
		{
			name: "force video rewrites zip metadata",
			request: attachmentPresignRequest{
				ConversationID: "c2c_user_1",
				FileName:       "holiday.zip",
				FileSize:       1024,
				ContentType:    "application/zip",
				IsVideo:        true,
			},
			wantFileName:    "holiday.mp4",
			wantContentType: "video/mp4",
			wantIsVideo:     true,
		},
		{
			name: "regular zip is unchanged",
			request: attachmentPresignRequest{
				ConversationID: "c2c_user_1",
				FileName:       "archive.zip",
				FileSize:       1024,
				ContentType:    "application/zip",
			},
			wantFileName:    "archive.zip",
			wantContentType: "application/zip",
			wantIsVideo:     false,
		},
		{
			name: "force video keeps known video extension",
			request: attachmentPresignRequest{
				ConversationID: "c2c_user_1",
				FileName:       "clip.mov",
				FileSize:       1024,
				ContentType:    "application/octet-stream",
				IsVideo:        true,
			},
			wantFileName:    "clip.mov",
			wantContentType: "video/quicktime",
			wantIsVideo:     true,
		},
	}

	settings := config.ChatAttachmentUploadSettings{MaxSizeMB: 300}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := normalizeAttachmentPresignRequest(tt.request, settings)
			if err != nil {
				t.Fatalf("normalizeAttachmentPresignRequest() error = %v", err)
			}
			if req.FileName != tt.wantFileName {
				t.Fatalf("FileName = %q, want %q", req.FileName, tt.wantFileName)
			}
			if req.ContentType != tt.wantContentType {
				t.Fatalf("ContentType = %q, want %q", req.ContentType, tt.wantContentType)
			}
			if req.IsVideo != tt.wantIsVideo {
				t.Fatalf("IsVideo = %v, want %v", req.IsVideo, tt.wantIsVideo)
			}
		})
	}
}
