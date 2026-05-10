package logger

import (
	"go.uber.org/zap"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		format  string
		output  string
		wantErr bool
	}{
		{
			name:    "debug json to stdout",
			level:   "debug",
			format:  "json",
			output:  "stdout",
			wantErr: false,
		},
		{
			name:    "info console to stdout",
			level:   "info",
			format:  "console",
			output:  "stdout",
			wantErr: false,
		},
		{
			name:    "warn json to stdout",
			level:   "warn",
			format:  "json",
			output:  "stdout",
			wantErr: false,
		},
		{
			name:    "error console to stdout",
			level:   "error",
			format:  "console",
			output:  "stdout",
			wantErr: false,
		},
		{
			name:    "invalid level defaults to info",
			level:   "invalid",
			format:  "json",
			output:  "stdout",
			wantErr: false,
		},
		{
			name:    "info to file",
			level:   "info",
			format:  "json",
			output:  "/tmp/test-log.log",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.level, tt.format, tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}

			if Log == nil {
				t.Error("Log should not be nil after Init")
			}
		})
	}
}

func TestLogFunctions(t *testing.T) {
	// Initialize logger first
	if err := Init("debug", "console", "stdout"); err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}

	t.Run("Debug", func(t *testing.T) {
		Debug("debug message")
		Debug("debug with field", zap.String("key", "value"))
	})

	t.Run("Info", func(t *testing.T) {
		Info("info message")
		Info("info with field", zap.String("key", "value"))
	})

	t.Run("Warn", func(t *testing.T) {
		Warn("warn message")
		Warn("warn with field", zap.String("key", "value"))
	})

	t.Run("Error", func(t *testing.T) {
		Error("error message")
		Error("error with field", zap.String("key", "value"))
	})

	t.Run("Sync", func(t *testing.T) {
		Sync()
	})

	t.Run("multiple fields", func(t *testing.T) {
		Info("multiple fields",
			zap.String("string", "value"),
			zap.Int("int", 123),
			zap.Bool("bool", true),
		)
	})
}

func TestMultipleInitCalls(t *testing.T) {
	// Test that multiple Init calls work properly
	for i := 0; i < 3; i++ {
		err := Init("info", "json", "stdout")
		if err != nil {
			t.Errorf("Init call %d failed: %v", i, err)
		}
	}
}