package utils

import (
	"io/fs"
	"os"
	os_user "os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FileManagerTestSuite struct {
	suite.Suite
	user   *os_user.User
	group  *os_user.Group
	tmpDir string
}

func (t *FileManagerTestSuite) SetupSuite() {
	currUser, err := os_user.Current()
	t.Require().NoError(err, "os/user.Current()")
	t.Require().NotNil(currUser, "currUser")
	t.user = currUser

	currGroup, err := os_user.LookupGroupId(currUser.Gid)
	t.Require().NoError(err, "os/user.LookupGroupId(currUser.Gid)")
	t.Require().NotNil(currGroup, "currGroup")
	t.group = currGroup
}

func (t *FileManagerTestSuite) TearDownSuite() {
}

func (t *FileManagerTestSuite) SetupTest() {
	tmpDir, err := os.MkdirTemp("", ".fileMgrTest.*")
	t.Require().NoError(err, "os.MkdirTemp()")
	t.Require().NotEmpty(tmpDir, "tmpDir should be setup")

	t.tmpDir = tmpDir
}

func (t *FileManagerTestSuite) TearDownTest() {
	err := os.RemoveAll(t.tmpDir)
	t.NoError(err, "os.RemoveAll(t.tmpDir)")
}

func (t *FileManagerTestSuite) SkipIfRoot() {
	if os.Geteuid() == 0 {
		t.T().Skipf("Test cannot be run as root")
	}
}

func (t *FileManagerTestSuite) Test_Paths() {
	var accessible bool

	// use a common test file path
	filename := filepath.Join(t.tmpDir, "test_file")

	t.False(CheckPathExists(filename), "test_file shouldn't exist yet")

	fm := NewManagedFile()
	defer fm.Close()

	// relative path
	err := fm.Init(
		"test_file",
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init() with relative path")

	// absolute path
	err = fm.Init(
		filename,
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init() with absolute path")

	err = fm.Create()
	t.NoError(err, "fm.Create() should work")

	t.True(CheckPathExists(filename), "test_file should exist now")

	err = fm.UseExistingFile(filename)
	t.NoError(err, "fm.UseExistingFile() should work")

	accessible, err = fm.Accessible()
	t.NoError(err, "accessibility check should have worked")
	t.True(accessible, "test_file should be accessible")

	err = os.Chmod(fm.Path(), 0000)
	t.NoError(err, "chmod'ing test_file file to be inaccessible")

	accessible, err = fm.Accessible()
	t.NoError(err, "accessibility check should have worked")
	t.False(accessible, "test_file shouldn't be accessible")
}

func (t *FileManagerTestSuite) Test_InitUserGroup() {
	// use a common test file path
	filename := filepath.Join(t.tmpDir, "test_file")

	// define tests table
	tests := []struct {
		name                   string
		path                   string
		user                   string
		group                  string
		perm                   os.FileMode
		expected_user_success  bool
		expected_username      string
		expected_username_msg  string
		expected_group_success bool
		expected_groupname     string
		expected_groupname_msg string
	}{
		{
			"default user and group",
			filename,
			"",
			"",
			0600,
			true,
			t.user.Username,
			"default user should be the current user",
			true,
			t.group.Name,
			"default group should be the current user's primary group",
		},
		{
			"root user and group by name",
			filename,
			"root",
			"root",
			0600,
			true,
			"root",
			"should be the root user",
			true,
			"root",
			"should be the root group",
		},
		{
			"root user and group by id",
			filename,
			"0",
			"0",
			0600,
			true,
			"root",
			"should be the root user",
			true,
			"root",
			"should be the root group",
		},
		{
			"unknown user should fail",
			filename,
			"unknown",
			"root",
			0600,
			false,
			"unknown",
			"should be an unknown user",
			true,
			"root",
			"should be the root group",
		},
		{
			"unknown group should fail",
			filename,
			"root",
			"unknown",
			0600,
			true,
			"root",
			"should be the root user",
			false,
			"unknown",
			"should be an unknown group",
		},
	}

	// run the test for each test table entry
	for _, tt := range tests {
		t.Run(
			"InitUserGroup "+tt.name,
			func() {
				fm := NewManagedFile()
				defer fm.Close()

				// test default user
				err := fm.Init(
					tt.path,
					tt.user,
					tt.group,
					tt.perm,
				)
				if !(tt.expected_user_success && tt.expected_group_success) {
					t.Error(err, "fm.Init() failed: %s", tt.name)
					return
				}

				t.NoError(err, "fm.Init()")

				t.NotEmpty(fm.user, "fm.user should be setup")
				t.Equal(tt.expected_username, fm.User(), tt.expected_username_msg)

				t.NotEmpty(fm.group, "fm.group should be setup")
				t.Equal(tt.expected_groupname, fm.Group(), tt.expected_groupname_msg)
			},
		)
	}
}

func (t *FileManagerTestSuite) Test_InitPerm() {
	fm := NewManagedFile()
	defer fm.Close()

	path := filepath.Join(t.tmpDir, "test_file")
	perm := os.FileMode(0600)

	// absolute path
	err := fm.Init(
		path,
		"",
		"",
		perm,
	)
	t.NoError(err, "fm.Init()")

	t.NotEmpty(fm.perm, "fm.perm should be setup")
	t.Equal(path, fm.Path(), "fm.Path() should be supplied perm")
	t.Equal(perm, fm.Perm(), "fm.Perm() should be supplied perm")
}

func (t *FileManagerTestSuite) Test_CreateCloseDelete() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create the file
	err = fm.Create()
	t.NoError(err, "create managed file")

	// created file should exist
	exists, err := fm.Exists()
	t.NoError(err, "created file should exist")
	t.True(exists, "created file should exist")

	// explicitly close the file
	err = fm.Close()
	t.NoError(err, "close previously created file")

	// delete the file
	err = fm.Delete()
	t.NoError(err, "delete the managed file")

	// created file should no longer exist
	exists, err = fm.Exists()
	t.NoError(err, "created file shouldn't exist")
	t.False(exists, "created file shouldn't exist")
}

func (t *FileManagerTestSuite) Test_CreateInvalidFilePath() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "does", "not", "exist"),
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create should fail because path is not valid
	err = fm.Create()
	t.Error(err, "invalid file path")
}

func (t *FileManagerTestSuite) Test_CreateInvalidUserName() {
	t.SkipIfRoot()

	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"root",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create should fail because we don't have privs to chown
	err = fm.Create()
	t.Error(err, "no privs to chown to specified user")
}

func (t *FileManagerTestSuite) Test_CreateInvalidUid() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"root",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create should fail because uid is not parseable
	fm.user.Uid = "not-a-number"
	err = fm.Create()
	t.Error(err, "can't chown to a non-parseable uid")
}

func (t *FileManagerTestSuite) Test_CreateInvalidGroupName() {
	t.SkipIfRoot()

	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"root",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create should fail because we don't have privs to chown
	err = fm.Create()
	t.Error(err, "invalid group")
}

func (t *FileManagerTestSuite) Test_CreateInvalidGid() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"root",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create should fail because gid is not parseable
	fm.group.Gid = "not-a-number"
	err = fm.Create()
	t.Error(err, "can't chown to a non-parseable gid")
}

func (t *FileManagerTestSuite) Test_InitNoAccessPerms() {
	fm := NewManagedFile()
	defer fm.Close()

	// create should fail as perms will deny file access
	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0,
	)
	t.Error(err, "no access perms specified")
}

func (t *FileManagerTestSuite) Test_CreateInvalidPerms() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// force perms to 0 and create should fail as perms will deny file access
	fm.perm = 0
	err = fm.Create()
	t.Error(err, "invalid perms specified")
}

func (t *FileManagerTestSuite) Test_UpdateDisabledAccessPerms() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create a valid file
	err = fm.Create()
	t.NoError(err, "created with valid perms")

	// attempt to update perms to remove access
	err = fm.SetPerm(0)
	t.Error(err, "no access perms specified")
}

func (t *FileManagerTestSuite) Test_CreateSpecialPerms() {
	t.SkipIfRoot()

	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0600|fs.ModeSetgid,
	)
	t.NoError(err, "fm.Init()'d")

	// attempt to create a file with special perms beyond access
	err = fm.Create()
	t.NoError(err, "create with special perms")
}

func (t *FileManagerTestSuite) Test_UpdateSpecialPerms() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create a valid file
	err = fm.Create()
	t.NoError(err, "created with valid perms")

	// attempt to update with special perms beyond access
	err = fm.SetPerm(fm.perm | fs.ModeSetgid)
	t.NoError(err, "update with special perms")
}

func (t *FileManagerTestSuite) Test_SetPath() {
	fm := NewManagedFile()
	defer fm.Close()

	patha := filepath.Join(t.tmpDir, "test_file_a")
	pathb := filepath.Join(t.tmpDir, "test_file_b")
	pathc := filepath.Join(t.tmpDir, "test_file_c")

	err := fm.Init(
		patha,
		"",
		"",
		0600,
	)
	t.NoError(err, "init first path")

	// first path shouldn't exist yet
	exists, err := fm.Exists()
	t.NoError(err, "first path Exists() shouldn't fail")
	t.False(exists, "first path shouldn't exist")
	t.Empty(fm.file, "first path not created/opened")

	// change the second path before creating the file
	err = fm.SetPath(pathb)
	t.NoError(err, "switch to second path")

	// second path shouldn't exist yet
	exists, err = fm.Exists()
	t.NoError(err, "second path Exists() shouldn't fail")
	t.False(exists, "second path shouldn't exist")
	t.Empty(fm.file, "second path not created/opened")

	// create the second file
	err = fm.Create()
	t.NoError(err, "create second path")

	// second path should now exist
	exists, err = fm.Exists()
	t.NoError(err, "second path Exists() shouldn't fail")
	t.True(exists, "second path should exist")
	t.NotEmpty(fm.file, "second path now created/opened")
	t.Equal(
		pathb,
		fm.file.Name(),
		"opened file name should match second file path",
	)

	// change to the third path
	err = fm.SetPath(pathc)
	t.NoError(err, "switch to third path")

	// third path shouldn't exist yet
	exists, err = fm.Exists()
	t.NoError(err, "third path Exists() shouldn't fail")
	t.False(exists, "third path shouldn't exist")
	t.Empty(fm.file, "third path not created/opened")

	// switch back to second path which should exist
	err = fm.SetPath(pathb)
	t.NoError(err, "switch back to second path")

	// confirm second path exists
	exists, err = fm.Exists()
	t.NoError(err, "second path Exists() shouldn't fail")
	t.True(exists, "second path should exist")
	t.Empty(fm.file, "second path yet opened")
}

func (t *FileManagerTestSuite) Test_ReadWriteBackup() {
	path := filepath.Join(t.tmpDir, "test_file")
	user := ""
	group := ""
	perm := os.FileMode(0600)
	first_body := []byte("the first content body")
	second_body := []byte("the second content body is longer")
	short_body := []byte("short body")

	// create a file manager for the target path
	orig := NewManagedFile()
	defer orig.Close()

	err := orig.Init(
		path,
		user,
		group,
		perm,
	)
	t.NoError(err, "orig.Init()d")

	// path shouldn't exist yet
	exists, err := orig.Exists()
	t.NoError(err, "orig.Exists() shouldn't fail")
	t.False(exists, "original path shouldn't exist")
	t.Empty(orig.file, "original path not created/opened")

	// create a file manager for the backup path
	bkupPath := orig.backupFileName()
	bkup := NewManagedFile()
	defer bkup.Close()

	err = bkup.Init(
		bkupPath,
		user,
		group,
		perm,
	)
	t.NoError(err, "bkup.Init()d")

	// bkupPath shouldn't exist yet
	exists, err = bkup.Exists()
	t.NoError(err, "bkup.Exists() shouldn't fail")
	t.False(exists, "backup path shouldn't exist")

	// backing up a file that doesn't exist yet should fail
	err = orig.Backup()
	t.Error(err, "original path doesn't exist yet")

	// create the file
	err = orig.Create()
	t.NoError(err, "orig.Create()")

	// original path should exist now
	exists, err = orig.Exists()
	t.NoError(err, "orig.Exists() shouldn't fail")
	t.True(exists, "original path should exist")
	t.NotEmpty(orig.file, "original path opened")

	// backups should be enabled by default
	t.True(orig.BackupsEnabled(), "backups should be enabled by default")

	// disable backups
	orig.DisableBackups()

	// backups should be disabled now
	t.False(orig.BackupsEnabled(), "backups should be disabled now")

	// backups are disabled so no backup should be created
	err = orig.Backup()
	t.NoError(err, "backup shouldn't fail when disabled")

	// bkupPath shouldn't exist yet
	exists, err = bkup.Exists()
	t.NoError(err, "bkup.Exists() shouldn't fail")
	t.False(exists, "backup path shouldn't exist yet")

	// disable backups
	orig.EnableBackups()

	// backups should be enabled again
	t.True(orig.BackupsEnabled(), "backups should be enabled again")

	// backup should now succeed
	err = orig.Backup()
	t.NoError(err, "backup of original path should succeed")

	// backup path should exist now
	exists, err = bkup.Exists()
	t.NoError(err, "bkup.Exists() shouldn't fail")
	t.True(exists, "backup path should exist")

	// write content to path
	err = orig.Update([]byte(first_body))
	t.NoError(err, "orig.Update(first_body) should succeed")

	// ensure path contents are match what was written
	content, err := orig.Read()
	t.NoError(err, "orig.Read() should succeed")
	t.Equal(first_body, content, "original content should match what was just written")

	// bkupPath should still exist
	exists, err = bkup.Exists()
	t.NoError(err, "bkup.Exists() shouldn't fail")
	t.True(exists, "bkupPath should exist")

	// open backup file
	err = bkup.Create()
	t.NoError(err, "should be able to open backup file")
	t.NotEmpty(bkup.file, "backup file should be open")

	// backup should be empty
	content, err = bkup.Read()
	t.NoError(err, "bkup.Read() should succeed")
	t.Empty(content, "backup content should empty")

	// close the backup
	err = bkup.Close()
	t.NoError(err, "bkup.Close() should succeed")

	// backup with initial non-empty content should succeed
	err = orig.Backup()
	t.NoError(err, "backup of initial non-empty content should succeed")

	// open backup file
	err = bkup.Create()
	t.NoError(err, "should be able to open backup file")
	t.NotEmpty(bkup.file, "backup file should be open")

	// backup contents should match initial non-empty content of original file
	content, err = bkup.Read()
	t.NoError(err, "bkup.Read() should succeed")
	t.Equal(first_body, content, "backup content should match original")

	// close the backup
	err = bkup.Close()
	t.NoError(err, "bkup.Close() should succeed")

	// update original with longer content
	err = orig.Update([]byte(second_body))
	t.NoError(err, "orig.Update(second_body) should succeed")

	// original content should have been updated
	content, err = orig.Read()
	t.NoError(err, "orig.Read() should succeed")
	t.Equal(second_body, content, "read content should match what was just written")

	// open backup file
	err = bkup.Create()
	t.NoError(err, "should be able to open backup file")
	t.NotEmpty(bkup.file, "backup file should be open")

	// backup content should not have changed
	content, err = bkup.Read()
	t.NoError(err, "bkup.Read() should succeed")
	t.Equal(first_body, content, "backup content should match initial update")

	// close the backup
	err = bkup.Close()
	t.NoError(err, "bkup.Close() should succeed")

	// update original with shorter content
	err = orig.Update([]byte(short_body))
	t.NoError(err, "orig.Update(short_body) should succeed")

	// original content should have been updated
	content, err = orig.Read()
	t.NoError(err, "orig.Read() should succeed")
	t.Equal(short_body, content, "original content should match what was just written")

	// backup with of short content should succeed
	err = orig.Backup()
	t.NoError(err, "backup of short content should succeed")

	// open backup file
	err = bkup.Create()
	t.NoError(err, "should be able to open backup file")
	t.NotEmpty(bkup.file, "backup file should be open")

	// backup content should not have changed
	content, err = bkup.Read()
	t.NoError(err, "bkup.Read() should succeed")
	t.Equal(short_body, content, "backup content should match short content")

	// close the backup
	err = bkup.Close()
	t.NoError(err, "bkup.Close() should succeed")
}

func (t *FileManagerTestSuite) Test_LockUnlock() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create the file
	err = fm.Create()
	t.NoError(err, "create the managed file")

	// file should be created as unlocked
	t.False(fm.IsLocked(), "managed file should be unlocked initially")

	// lock the file
	err = fm.Lock()
	t.NoError(err, "locking managed file")

	// file should be marked as locked
	t.True(fm.IsLocked(), "managed file should now be locked")

	// unlock the file
	err = fm.Unlock()
	t.NoError(err, "unlocking managed file")

	// file should be marked as locked
	t.False(fm.IsLocked(), "managed file should now be unlocked")
}

func (t *FileManagerTestSuite) Test_CloseLockedFile() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create the file
	err = fm.Create()
	t.NoError(err, "create the managed file")

	// file should be created as unlocked
	t.False(fm.IsLocked(), "managed file should be unlocked initially")

	// lock the file
	err = fm.Lock()
	t.NoError(err, "locking managed file")

	// file should be marked as locked
	t.True(fm.IsLocked(), "managed file should now be locked")

	// close the file
	err = fm.Close()
	t.NoError(err, "closing locked managed file")

	// file should be marked as locked
	t.False(fm.IsLocked(), "managed file should now be unlocked")
}

func (t *FileManagerTestSuite) Test_DeleteLockedFile() {
	fm := NewManagedFile()
	defer fm.Close()

	err := fm.Init(
		filepath.Join(t.tmpDir, "test_file"),
		"",
		"",
		0600,
	)
	t.NoError(err, "fm.Init()'d")

	// create the file
	err = fm.Create()
	t.NoError(err, "create the managed file")

	// file should be created as unlocked
	t.False(fm.IsLocked(), "managed file should be unlocked initially")

	// lock the file
	err = fm.Lock()
	t.NoError(err, "locking managed file")

	// file should be marked as locked
	t.True(fm.IsLocked(), "managed file should now be locked")

	// unlock the file
	err = fm.Delete()
	t.NoError(err, "deleting locked managed file")

	// file should be marked as locked
	t.False(fm.IsLocked(), "managed file should now be unlocked")
}

func TestFileManager(t *testing.T) {
	suite.Run(t, new(FileManagerTestSuite))
}
