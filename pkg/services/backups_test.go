// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/datastore"
	"github.com/James-HoneyBadger/honey-ledger/pkg/settings"
)

func TestIsPathWithinDirectory(t *testing.T) {
	baseDir := filepath.Join(string(filepath.Separator), "data", "storage")

	assert.True(t, isPathWithinDirectory(baseDir, filepath.Join(baseDir, "avatars", "1.png")))
	assert.False(t, isPathWithinDirectory(baseDir, filepath.Join(string(filepath.Separator), "data", "storage2", "1.png")))
	assert.False(t, isPathWithinDirectory(baseDir, filepath.Join(baseDir, "..", "secret")))
	assert.False(t, isPathWithinDirectory(baseDir, baseDir))
}

func TestResolveBackupPath_RejectsTraversal(t *testing.T) {
	_, err := resolveBackupPath(filepath.Join(string(filepath.Separator), "data", "backups"), ".."+string(filepath.Separator)+"backup_20240101_000000.zip")

	assert.Error(t, err)
}

func TestResolveBackupPath_AcceptsValidName(t *testing.T) {
	backupDir := filepath.Join(string(filepath.Separator), "data", "backups")
	actual, err := resolveBackupPath(backupDir, "backup_20240101_000000.zip")

	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(backupDir, "backup_20240101_000000.zip"), actual)
}

func TestCreateAndRestoreBackup_SQLite(t *testing.T) {
	workingDir := t.TempDir()
	databaseDir := filepath.Join(workingDir, "data")
	storageDir := filepath.Join(workingDir, "storage")

	assert.Nil(t, os.MkdirAll(databaseDir, 0750))
	assert.Nil(t, os.MkdirAll(storageDir, 0750))

	config := &settings.Config{
		WorkingPath: workingDir,
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:      settings.Sqlite3DbType,
			DatabasePath:      filepath.Join(databaseDir, "honey-ledger.db"),
			MaxIdleConnection: 1,
			MaxOpenConnection: 1,
		},
		LocalFileSystemPath: storageDir,
	}

	previousConfig := settings.Container.GetCurrentConfig()
	settings.SetCurrentConfig(config)
	assert.Nil(t, datastore.InitializeDataStore(config))

	defer func() {
		_ = datastore.Container.Close()
		settings.SetCurrentConfig(previousConfig)
	}()

	ctx := core.NewNullContext()
	database := datastore.Container.UserDataStore.Choose(0)

	assert.Nil(t, database.Exec(ctx, "CREATE TABLE backup_test (id INTEGER PRIMARY KEY, value TEXT NOT NULL)"))
	assert.Nil(t, database.Exec(ctx, "INSERT INTO backup_test (id, value) VALUES (1, 'before')"))

	storageFilePath := filepath.Join(storageDir, "sample.txt")
	assert.Nil(t, os.WriteFile(storageFilePath, []byte("payload"), 0600))

	backup, err := Backups.CreateBackup(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, backup)

	assert.Nil(t, database.Exec(ctx, "UPDATE backup_test SET value = 'after' WHERE id = 1"))
	assert.Nil(t, os.WriteFile(storageFilePath, []byte("changed"), 0600))

	assert.Nil(t, Backups.RestoreBackup(ctx, backup.FileName))

	rows, err := datastore.Container.UserDataStore.Choose(0).NewSession(ctx).QueryString("SELECT value FROM backup_test WHERE id = ?", 1)
	assert.Nil(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "before", rows[0]["value"])

	restoredStorageContent, err := os.ReadFile(storageFilePath)
	assert.Nil(t, err)
	assert.Equal(t, "payload", string(restoredStorageContent))
}

// TestRestoreBackup_CorruptDatabaseLeavesLiveDatabaseIntact restores an archive
// whose database entry is not a SQLite file and verifies the live database is
// untouched and still usable.
func TestRestoreBackup_CorruptDatabaseLeavesLiveDatabaseIntact(t *testing.T) {
	workingDir := t.TempDir()
	databaseDir := filepath.Join(workingDir, "data")

	assert.Nil(t, os.MkdirAll(databaseDir, 0750))

	databasePath := filepath.Join(databaseDir, "honey-ledger.db")
	config := &settings.Config{
		WorkingPath: workingDir,
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:      settings.Sqlite3DbType,
			DatabasePath:      databasePath,
			MaxIdleConnection: 1,
			MaxOpenConnection: 1,
		},
	}

	previousConfig := settings.Container.GetCurrentConfig()
	settings.SetCurrentConfig(config)
	assert.Nil(t, datastore.InitializeDataStore(config))

	defer func() {
		_ = datastore.Container.Close()
		settings.SetCurrentConfig(previousConfig)
	}()

	ctx := core.NewNullContext()
	assert.Nil(t, datastore.Container.UserDataStore.Choose(0).Exec(ctx, "CREATE TABLE backup_test (id INTEGER PRIMARY KEY)"))

	backupPath := filepath.Join(workingDir, "backups", "backup_20240101_000000.zip")
	assert.Nil(t, os.MkdirAll(filepath.Dir(backupPath), 0750))

	archive, err := os.Create(backupPath)
	assert.Nil(t, err)

	zipWriter := zip.NewWriter(archive)
	databaseEntry, err := zipWriter.Create("database/honey-ledger.db")
	assert.Nil(t, err)
	_, err = databaseEntry.Write([]byte("this is not a sqlite database"))
	assert.Nil(t, err)
	assert.Nil(t, zipWriter.Close())
	assert.Nil(t, archive.Close())

	before, err := os.ReadFile(databasePath)
	assert.Nil(t, err)
	assert.True(t, bytes.HasPrefix(before, sqliteFileHeader))

	err = Backups.RestoreBackup(ctx, "backup_20240101_000000.zip")
	assert.Error(t, err)

	after, err := os.ReadFile(databasePath)
	assert.Nil(t, err)
	assert.True(t, bytes.HasPrefix(after, sqliteFileHeader), "the live database keeps its SQLite header after a failed restore")

	rows, err := datastore.Container.UserDataStore.Choose(0).NewSession(ctx).QueryString("SELECT name FROM sqlite_master WHERE type='table' AND name='backup_test'")
	assert.Nil(t, err)
	assert.Len(t, rows, 1, "the live database stays usable after a failed restore")
}

func TestAtomicReplaceFile(t *testing.T) {
	workingDir := t.TempDir()
	destPath := filepath.Join(workingDir, "target.db")
	assert.Nil(t, os.WriteFile(destPath, []byte("original"), 0600))

	validate := func(file *os.File) error {
		header := make([]byte, len(sqliteFileHeader))

		if _, err := file.ReadAt(header, 0); err != nil {
			return err
		}

		if !bytes.Equal(header, sqliteFileHeader) {
			return fmt.Errorf("not a sqlite file")
		}

		return nil
	}

	err := atomicReplaceFile(strings.NewReader("corrupt"), destPath, validate)
	assert.Error(t, err)

	content, err := os.ReadFile(destPath)
	assert.Nil(t, err)
	assert.Equal(t, "original", string(content), "a rejected replacement must leave the file untouched")

	valid := append(append([]byte{}, sqliteFileHeader...), []byte("payload")...)
	assert.Nil(t, atomicReplaceFile(bytes.NewReader(valid), destPath, validate))

	content, err = os.ReadFile(destPath)
	assert.Nil(t, err)
	assert.Equal(t, valid, content)

	entries, err := os.ReadDir(workingDir)
	assert.Nil(t, err)
	assert.Len(t, entries, 1, "no temporary or backup files are left behind")

	info, err := os.Stat(destPath)
	assert.Nil(t, err)

	if runtime.GOOS != "windows" {
		// Windows has no POSIX permission bits; the mode it reports carries
		// only the read-only bit, so the private-permission guarantee is
		// checked where the bits exist.
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "restored files keep private permissions")
	}
}
