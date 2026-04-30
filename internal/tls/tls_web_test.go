package tls

import (
	"testing"
)

func TestTrimKey(t *testing.T) {
	tests := []struct {
		name                string
		key                 []byte
		want                string
		expectErr           bool
		leadingZeroedBytes  int
		trailingZeroedBytes int
	}{
		{
			name: "all valid chars",
			key:  []byte("myexcellentthing"),
			want: "myexcellentthing",
		},
		{
			name:                "whitespace in middle",
			key:                 []byte("myexcellentthing 12345"),
			want:                "myexcellentthing",
			trailingZeroedBytes: 6,
		},
		{
			name:                "new line in middle",
			key:                 []byte("myexcellentthing\n12345"),
			want:                "myexcellentthing",
			trailingZeroedBytes: 6,
		},
		{
			name:                "end with null char",
			key:                 append([]byte("my@!()0187212345"), 0),
			want:                "my@!()0187212345",
			trailingZeroedBytes: 1,
		},
		{
			name:                "start end and middle bad chars",
			key:                 []byte(" myexcellentthing\n12345 "),
			want:                "myexcellentthing",
			leadingZeroedBytes:  1,
			trailingZeroedBytes: 7,
		},
		{
			name:                "start and middle bad chars",
			key:                 []byte("\x00myexcellentthing\n12345"),
			want:                "myexcellentthing",
			leadingZeroedBytes:  1,
			trailingZeroedBytes: 6,
		},
		{
			name:                "start end and middle bad chars (null byte)",
			key:                 []byte(" myexcellentthing\x0012345 "),
			want:                "myexcellentthing",
			leadingZeroedBytes:  1,
			trailingZeroedBytes: 7,
		},
		{
			name:                "start and end bad chars",
			key:                 []byte(" myexcellentthing12345 "),
			want:                "myexcellentthing12345",
			leadingZeroedBytes:  1,
			trailingZeroedBytes: 1,
		},
		{
			name: "single char",
			key:  []byte("!"),
			want: "!",
		},
		{
			name:      "no chars",
			key:       []byte(""),
			want:      "",
			expectErr: true,
		},
		{
			name:      "all bad chars",
			key:       []byte("   "),
			want:      "   ",
			expectErr: true,
		},
		{
			name:      "mixed bad chars",
			key:       append([]byte("\n  \n \n"), 0),
			want:      "   ",
			expectErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := trimKey(test.key)
			if test.expectErr {
				if got != nil {
					t.Errorf("trimKey() = %v, want nil", got)
				}
				// Check that all bytes of test.key are zeroed
				for i, b := range test.key {
					if b != 0 {
						t.Errorf("trimKey() did not zero out byte %d, value: %v", i, b)
					}
				}
				return
			}
			if got.String() != test.want {
				t.Errorf("trimKey() = %v, want %v", got.String(), test.want)
			}
			// Check that the correct number of bytes are zeroed
			for i := 0; i < test.leadingZeroedBytes; i++ {
				b := test.key[i]
				if test.key[i] != 0 {
					t.Errorf("trimKey() did not zero out byte %d, value: %v", i, b)
				}
			}
			for i := 0; i < test.trailingZeroedBytes; i++ {
				b := test.key[len(test.key)-i-1]
				if b != 0 {
					t.Errorf("trimKey() did not zero out byte %d, value: %v", i, b)
				}
			}
		})
	}
}
