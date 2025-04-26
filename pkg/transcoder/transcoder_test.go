package transcoder

import (
	"testing"

	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/progress"
)

// mockProgressReporter simple mock
type mockProgressReporter struct{}

func (m *mockProgressReporter) Start(total int64)                 {}
func (m *mockProgressReporter) Update(current int64, _, _ string) {}
func (m *mockProgressReporter) Increment(_, _ string)             {}
func (m *mockProgressReporter) Complete()                         {}
func (m *mockProgressReporter) Updates() <-chan progress.ProgressEvent {
	ch := make(chan progress.ProgressEvent)
	close(ch)
	return ch
}
func (m *mockProgressReporter) Close()                {}
func (m *mockProgressReporter) JSON() (string, error) { return "{}", nil }

func TestNewTranscoderValidation(t *testing.T) {
	mockReporter := &mockProgressReporter{}

	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name: "Valid Options",
			opts: Options{
				InputPath:  "input.mp4",
				OutputPath: "output/dir",
			},
			wantErr: false,
		},
		{
			name: "Missing Input Path",
			opts: Options{
				OutputPath: "output/dir",
			},
			wantErr: true,
		},
		{
			name: "Missing Output Path",
			opts: Options{
				InputPath: "input.mp4",
			},
			wantErr: true,
		},
		{
			name: "Remote without Downloader (using New)", // New creates a default downloader if needed
			opts: Options{
				InputPath:     "http://example.com/video.mp4",
				IsRemoteInput: true,
				OutputPath:    "output/dir",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_New", func(t *testing.T) {
			_, err := New(tt.opts, mockReporter)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr {
				if _, ok := err.(*errors.StructuredError); !ok {
					t.Errorf("New() returned non-structured error: %T", err)
				}
			}
		})
		t.Run(tt.name+"_NewWithDeps", func(t *testing.T) {
			_, err := NewWithDeps(tt.opts, mockReporter, nil, nil) // Pass nil logger/downloader
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWithDeps() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr {
				if _, ok := err.(*errors.StructuredError); !ok {
					t.Errorf("NewWithDeps() returned non-structured error: %T", err)
				}
			}
		})
	}
}

func TestNewTranscoderDefaults(t *testing.T) {
	mockReporter := &mockProgressReporter{}
	opts := Options{InputPath: "in", OutputPath: "out"} // Minimum valid options

	trans, err := New(opts, mockReporter)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if trans.options.OutputType != HLSOutput {
		t.Errorf("Default OutputType: got %q, want %q", trans.options.OutputType, HLSOutput)
	}
	if trans.options.FFmpegBinary != "ffmpeg" {
		t.Errorf("Default FFmpegBinary: got %q, want %q", trans.options.FFmpegBinary, "ffmpeg")
	}
	if trans.options.DownloadDir != "downloads" {
		t.Errorf("Default DownloadDir: got %q, want %q", trans.options.DownloadDir, "downloads")
	}

	// Test that NewWithDeps also sets defaults
	transDeps, errDeps := NewWithDeps(opts, mockReporter, nil, nil)
	if errDeps != nil {
		t.Fatalf("NewWithDeps() failed: %v", errDeps)
	}

	if transDeps.options.OutputType != HLSOutput {
		t.Errorf("NewWithDeps Default OutputType: got %q, want %q", transDeps.options.OutputType, HLSOutput)
	}
	if transDeps.options.FFmpegBinary != "ffmpeg" {
		t.Errorf("NewWithDeps Default FFmpegBinary: got %q, want %q", transDeps.options.FFmpegBinary, "ffmpeg")
	}
	if transDeps.options.DownloadDir != "downloads" {
		t.Errorf("NewWithDeps Default DownloadDir: got %q, want %q", transDeps.options.DownloadDir, "downloads")
	}
}
