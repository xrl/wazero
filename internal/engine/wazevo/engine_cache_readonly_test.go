package wazevo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/tetratelabs/wazero/internal/filecache"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

func TestReadOnlyCacheStopsBeforeGuestCompilation(t *testing.T) {
	// This deliberately incomplete module panics if the guest compiler is entered.
	// No compiler machinery is initialized in the engine below.
	module := &wasm.Module{FunctionSection: []wasm.Index{0}, CodeSection: []wasm.Code{{Body: []byte{0xff}}}}
	t.Run("compiler poison is effective", func(t *testing.T) {
		defer func() { require.True(t, recover() != nil) }()
		_, _ = (&engine{}).compileModule(context.Background(), module, nil, false)
	})
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{"missing", nil, filecache.ErrMiss},
		{"stale", concat(magic, []byte{byte(len(testVersion))}, []byte("9.9.9"), make([]byte, 4)), filecache.ErrStale},
		{"corrupt", []byte("broken"), filecache.ErrCorrupt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			rw := filecache.New(dir)
			if tc.data != nil {
				require.NoError(t, rw.Add(fileCacheKey(module), bytes.NewReader(tc.data)))
			}
			before := snapshotCacheFiles(t, dir)
			e := &engine{fileCache: filecache.NewReadOnly(dir), wazeroVersion: testVersion}
			err := e.CompileModule(context.Background(), module, nil, false)
			require.True(t, errors.Is(err, tc.want), err)
			require.Equal(t, before, snapshotCacheFiles(t, dir))
			require.Equal(t, 0, len(e.compiledModules))
		})
	}
}

func snapshotCacheFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	ret := map[string]string{}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err)
		info, err := entry.Info()
		require.NoError(t, err)
		ret[entry.Name()] = string(data) + info.ModTime().String() + info.Mode().String()
	}
	return ret
}

func TestDeserializeCompiledModuleReleasesMapping(t *testing.T) {
	cm := &compiledModule{
		executables:     &executables{executable: []byte{1, 2, 3, 4}},
		functionOffsets: []int{0},
	}
	data, err := io.ReadAll(serializeCompiledModule(testVersion, cm))
	require.NoError(t, err)
	// Every truncation after the mapping is allocated must unmap exactly once,
	// including the stale old-format path with no catch-clause table.
	codeStart := len(magic) + 1 + len(testVersion) + 4 + 8 + 8
	for end := codeStart; end < len(data); end++ {
		t.Run(string(rune('a'+end-codeStart)), func(t *testing.T) {
			unmapped := 0
			got, stale, err := deserializeCompiledModuleWithUnmap(testVersion, io.NopCloser(bytes.NewReader(data[:end])), func(b []byte) error {
				unmapped++
				require.Equal(t, 4, len(b))
				return platform.MunmapCodeSegment(b)
			})
			require.Nil(t, got)
			require.True(t, err != nil || stale)
			require.Equal(t, 1, unmapped)
		})
	}
	t.Run("checksum mismatch", func(t *testing.T) {
		corrupt := bytes.Clone(data)
		corrupt[codeStart] ^= 0xff
		unmapped := 0
		_, _, err := deserializeCompiledModuleWithUnmap(testVersion, io.NopCloser(bytes.NewReader(corrupt)), func(b []byte) error {
			unmapped++
			return platform.MunmapCodeSegment(b)
		})
		require.Error(t, err)
		require.Equal(t, 1, unmapped)
	})
	t.Run("successful load transfers ownership", func(t *testing.T) {
		got, stale, err := deserializeCompiledModuleWithUnmap(testVersion, io.NopCloser(bytes.NewReader(data)), func([]byte) error {
			t.Fatal("unexpected unmap")
			return nil
		})
		require.NoError(t, err)
		require.False(t, stale)
		require.NoError(t, platform.MunmapCodeSegment(got.executable))
	})
}

func TestDeserializeCompiledModuleEmptyExecutable(t *testing.T) {
	cm := &compiledModule{executables: &executables{}}
	got, stale, err := deserializeCompiledModule(testVersion, io.NopCloser(serializeCompiledModule(testVersion, cm)))
	require.NoError(t, err)
	require.False(t, stale)
	require.Equal(t, 0, len(got.executable))
}
