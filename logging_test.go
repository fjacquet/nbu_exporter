package main

// func TestInfoLogger(t *testing.T) {
// 	var buf bytes.Buffer
// 	log.SetOutput(&buf)
// 	defer func() {
// 		log.SetOutput(os.Stderr)
// 	}()

// 	tests := []struct {
// 		name     string
// 		msg      string
// 		expected string
// 	}{
// 		{
// 			name:     "log info message",
// 			msg:      "This is an info message",
// 			expected: "level=info msg=\"This is an info message\" job=test_program\n",
// 		},
// 		{
// 			name:     "log empty message",
// 			msg:      "",
// 			expected: "level=info msg=\"\" job=test_program\n",
// 		},
// 		{
// 			name:     "log message with special characters",
// 			msg:      "This is a message with special characters: !@#$%^&*()",
// 			expected: "level=info msg=\"This is a message with special characters: !@#$%^&*()\" job=test_program\n",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			buf.Reset()
// 			programName = "test_program"
// 			InfoLogger(tt.msg)
// 			if buf.String() != tt.expected {
// 				t.Errorf("InfoLogger(%q) = %q, want %q", tt.msg, buf.String(), tt.expected)
// 			}
// 		})
// 	}
// }

// func TestPanicLoggerErr(t *testing.T) {
// 	t.Run("panic on error", func(t *testing.T) {
// 		defer func() {
// 			if r := recover(); r == nil {
// 				t.Errorf("PanicLoggerErr did not panic")
// 			}
// 		}()

// 		err := fmt.Errorf("test error")
// 		PanicLoggerErr(err)
// 	})

// 	t.Run("no panic on nil error", func(t *testing.T) {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				t.Errorf("PanicLoggerErr panicked unexpectedly")
// 			}
// 		}()

// 		PanicLoggerErr(nil)
// 	})
// }

// func TestPanicLoggerStr(t *testing.T) {
// 	t.Run("panic with valid message", func(t *testing.T) {
// 		defer func() {
// 			if r := recover(); r == nil {
// 				t.Errorf("Expected panic, but did not panic")
// 			}
// 		}()

// 		PanicLoggerStr("test panic message")
// 	})

// 	t.Run("panic with empty message", func(t *testing.T) {
// 		defer func() {
// 			if r := recover(); r == nil {
// 				t.Errorf("Expected panic, but did not panic")
// 			}
// 		}()

// 		PanicLoggerStr("")
// 	})
// }

// func TestProcessError(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		err      error
// 		wantExit bool
// 	}{
// 		{
// 			name:     "nil error",
// 			err:      nil,
// 			wantExit: false,
// 		},
// 		{
// 			name:     "non-nil error",
// 			err:      fmt.Errorf("test error"),
// 			wantExit: true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			old := os.Stdout
// 			r, w, _ := os.Pipe()
// 			os.Stdout = w

// 			exitCalled := false
// 			defer func() {
// 				if r := recover(); r != nil {
// 					exitCalled = true
// 				}
// 				os.Stdout = old
// 				w.Close()
// 				r.Close()
// 			}()

// 			ProcessError(tt.err)

// 			if exitCalled != tt.wantExit {
// 				t.Errorf("ProcessError() exit called = %v, want %v", exitCalled, tt.wantExit)
// 			}

// 			if tt.err != nil {
// 				buf := new(bytes.Buffer)
// 				buf.ReadFrom(r)
// 				got := buf.String()
// 				want := tt.err.Error() + "\n"
// 				if got != want {
// 					t.Errorf("ProcessError() output = %q, want %q", got, want)
// 				}
// 			}
// 		})
// 	}
// }
