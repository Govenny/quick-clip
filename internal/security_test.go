package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testContentKey = "11112222111122221111222211112222"

func TestSaveContentCreatesEncryptedPrimary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.json")
	want := []any{map[string]any{"command": "echo hello\necho world"}}

	if err := SaveContent(path, testContentKey, want); err != nil {
		t.Fatalf("SaveContent() error = %v", err)
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(onDisk) == "[{\"command\":\"echo hello\\necho world\"}]" {
		t.Fatal("primary file was written as plaintext JSON")
	}

	got, err := ReadContent(path, testContentKey)
	if err != nil {
		t.Fatalf("ReadContent() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadContent() = %#v, want %#v", got, want)
	}
}

func TestReadContentRecoversCorruptPrimaryFromBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.json")
	first := []any{map[string]any{"command": "first version"}}
	second := []any{map[string]any{"command": "second version"}}

	if err := SaveContent(path, testContentKey, first); err != nil {
		t.Fatalf("first SaveContent() error = %v", err)
	}
	if err := SaveContent(path, testContentKey, second); err != nil {
		t.Fatalf("second SaveContent() error = %v", err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("backup file was not created: %v", err)
	}

	if err := os.WriteFile(path, []byte("corrupt-primary"), 0600); err != nil {
		t.Fatalf("corrupt primary: %v", err)
	}

	got, err := ReadContent(path, testContentKey)
	if err != nil {
		t.Fatalf("ReadContent() recovery error = %v", err)
	}
	if !reflect.DeepEqual(got, first) {
		t.Fatalf("ReadContent() recovered %#v, want %#v", got, first)
	}

	restored, err := ReadContent(path, testContentKey)
	if err != nil {
		t.Fatalf("ReadContent() after restoration error = %v", err)
	}
	if !reflect.DeepEqual(restored, first) {
		t.Fatalf("restored primary = %#v, want %#v", restored, first)
	}
}

func TestReadContentRejectsCorruptPrimaryAndBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.json")
	if err := os.WriteFile(path, []byte("corrupt-primary"), 0600); err != nil {
		t.Fatalf("write corrupt primary: %v", err)
	}
	if err := os.WriteFile(path+".bak", []byte("corrupt-backup"), 0600); err != nil {
		t.Fatalf("write corrupt backup: %v", err)
	}

	got, err := ReadContent(path, testContentKey)
	if err == nil {
		t.Fatalf("ReadContent() = %#v, want error", got)
	}
	if got != nil {
		t.Fatalf("ReadContent() data = %#v, want nil on unrecoverable files", got)
	}
}
