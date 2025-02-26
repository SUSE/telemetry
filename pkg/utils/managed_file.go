package utils

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	os_user "os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

func getUserInfo(u string) (user *os_user.User, err error) {
	// use current user if no user specified
	if u == "" {
		return os_user.Current()
	}

	// first attempt to lookup user by name
	user, lname_err := os_user.Lookup(u)
	if lname_err == nil {
		return
	}

	slog.Debug(
		"Lookup() failed",
		slog.String("user", u),
		slog.String("err", lname_err.Error()),
	)

	// next try to lookup user by id
	user, lid_err := os_user.LookupId(u)
	if lid_err == nil {
		return
	}

	slog.Debug(
		"LookupId() failed",
		slog.String("uid", u),
		slog.String("err", lid_err.Error()),
	)

	err = fmt.Errorf("failed to retrieve user by name (%w) or id (%w)", lname_err, lid_err)
	return
}

func getGroupInfo(g string) (group *os_user.Group, err error) {
	skip_group_lookup := false
	var lname_err, lid_err error

	// use current user's primary group if no group specified
	if g == "" {
		user, err := os_user.Current()
		if err != nil {
			return nil, err
		}

		g = user.Gid

		// using primary gid so skip lookup by group name
		skip_group_lookup = true
	}

	if !skip_group_lookup {
		// first attempt to lookup group by name
		group, lname_err = os_user.LookupGroup(g)
		if lname_err == nil {
			return
		}

		slog.Debug(
			"LookupGroup() failed",
			slog.String("group", g),
			slog.String("err", lname_err.Error()),
		)
	}

	// next try to lookup group by id
	group, lid_err = os_user.LookupGroupId(g)
	if lid_err == nil {
		return
	}

	slog.Debug(
		"LookupGroupId() failed",
		slog.String("gid", g),
		slog.String("err", lid_err.Error()),
	)

	err = fmt.Errorf("failed to retrieve group by name (%w) or id (%w)", lname_err, lid_err)
	return
}

// mockable os interface
type mockableOs interface {
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
	Remove(name string) error
	Stat(name string) (os.FileInfo, error)
}

type realOs struct{}

// call real os.OpenFile()
func (r *realOs) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// call real os.Remove()
func (r *realOs) Remove(name string) error {
	return os.Remove(name)
}

// call real os.Stat()
func (r *realOs) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

var _ mockableOs = (*realOs)(nil)

// mockable syscall interface
type mockableSyscall interface {
	Flock(fd int, how int) (err error)
}

type realSyscall struct{}

func (r *realSyscall) Flock(fd int, how int) (err error) {
	return syscall.Flock(fd, how)
}

var _ mockableSyscall = (*realSyscall)(nil)

// FileManager interface
type FileManager interface {
	// init the structure
	Init(path string, owner string, group string, perm os.FileMode) error

	// file path
	Path() string
	SetPath(path string) error
	Exists() (bool, error)

	// file ownership and permissions
	User() string
	SetUser(string) error
	Group() string
	SetGroup(string) error
	Perm() os.FileMode
	SetPerm(perm os.FileMode) error

	// lock management
	Lock() error
	Unlock() error
	IsLocked() bool

	// data operations
	Create() error
	Read() ([]byte, error)
	Update(updatedContent []byte) error
	Backup() error
	Delete() error
	Close() error
}

// ManagedFile
type ManagedFile struct {
	os      mockableOs      // normally the os module, replaceable for testing purposes
	syscall mockableSyscall // normally the syscall module, replaceable for testing purposes
	path    string
	user    *os_user.User
	group   *os_user.Group
	perm    os.FileMode
	locked  bool
	file    *os.File
}

func NewManagedFile() *ManagedFile {
	return &ManagedFile{
		os:      &realOs{},
		syscall: &realSyscall{},
	}
}

func (fm *ManagedFile) dbg(msg string) {
	slog.Debug(
		msg,
		slog.String("path", fm.path),
	)
}

func (fm *ManagedFile) dbg_with_err(msg string, err error) error {
	slog.Debug(
		msg,
		slog.String("path", fm.path),
		slog.String("err", err.Error()),
	)

	return err
}

func (fm *ManagedFile) err_with_err(msg string, err error) error {
	slog.Error(
		msg,
		slog.String("path", fm.path),
		slog.String("err", err.Error()),
	)

	return err
}

func (fm *ManagedFile) checkPath() (err error) {
	if fm.path == "" {
		err = fmt.Errorf("managed file path not setup")
		fm.dbg_with_err("managed file not setup", err)
	}
	return
}

func (fm *ManagedFile) Init(path, user, group string, perm os.FileMode) (err error) {
	if err = fm.SetPerm(perm); err != nil {
		return
	}
	if err = fm.SetUser(user); err != nil {
		return
	}
	if err = fm.SetGroup(group); err != nil {
		return
	}
	if err = fm.SetPath(path); err != nil {
		return
	}
	fm.locked = false

	return
}

func (fm *ManagedFile) Path() string {
	return fm.path
}

func (fm *ManagedFile) SetPath(path string) (err error) {
	if !filepath.IsAbs(path) {
		return fm.dbg_with_err(
			fmt.Sprintf("non-absolute path: %q", path),
			fmt.Errorf("managed file paths must be absolute"),
		)
	}

	if fm.file != nil {
		if err = fm.Close(); err != nil {
			return fm.err_with_err("failed to close existing file", err)
		}
	}

	fm.path = path
	return
}

func (fm *ManagedFile) Exists() (exists bool, err error) {
	if err = fm.checkPath(); err != nil {
		return
	}

	fm.dbg("checking if managed file exists")

	if _, err = fm.os.Stat(fm.path); err == nil {
		// stat succeeded so file exists
		exists = true
	} else if errors.Is(err, fs.ErrNotExist) {
		// stat failed because file doesn't exists
		err = nil
	} else {
		fm.dbg_with_err("failed to stat file", err)
	}

	return
}

// idStr2int32() parses a uid or gid string to an int32
func idStr2int32(idName, idStr string) (res int32, err error) {
	// the idStr should be a base 10 representation of an int32 value
	parsed, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		slog.Debug(
			fmt.Sprintf("failed to parse %s as an int", idName),
			slog.String(idName, idStr),
			slog.String("err", err.Error()),
		)
		err = fmt.Errorf(
			"failed to parse %v %q as a 32-bit int: %w",
			idName,
			idStr,
			err,
		)
		return
	}

	res = int32(parsed)

	return
}

func (fm *ManagedFile) chownFile(file *os.File) (err error) {
	path := file.Name()

	slog.Debug(
		"chown()ing file",
		slog.String("path", path),
		slog.String("uid", fm.user.Uid),
		slog.String("gid", fm.group.Gid),
	)

	uid, err := idStr2int32("uid", fm.user.Uid)
	if err != nil {
		return
	}

	gid, err := idStr2int32("gid", fm.group.Gid)
	if err != nil {
		return
	}

	if err = file.Chown(int(uid), int(gid)); err != nil {
		slog.Error(
			"chown() failed",
			slog.String("path", path),
			slog.String("uid", fm.user.Uid),
			slog.String("gid", fm.group.Gid),
			slog.String("err", err.Error()),
		)
		err = fmt.Errorf(
			"chown(%q, %v, %v) failed: %w",
			path,
			fm.user.Uid,
			fm.group.Gid,
			err,
		)
	}

	return
}

func (fm *ManagedFile) chown() (err error) {
	// can't update existing file ownership if we don't have
	// an open file, and user and group info
	if (fm.file == nil) || (fm.user == nil) || (fm.group == nil) {
		return
	}

	// chown the opened file
	return fm.chownFile(fm.file)
}

func (fm *ManagedFile) User() string {
	return fm.user.Username
}

func (fm *ManagedFile) SetUser(user string) (err error) {
	if fm.user, err = getUserInfo(user); err != nil {
		slog.Error(
			"failed to get user info",
			slog.String("user", user),
			slog.String("err", err.Error()),
		)
		return
	}

	// update ownership of existing file if appropriate
	return fm.chown()
}

func (fm *ManagedFile) Group() string {
	return fm.group.Name
}

func (fm *ManagedFile) SetGroup(group string) (err error) {
	if fm.group, err = getGroupInfo(group); err != nil {
		slog.Error(
			"failed to get group info",
			slog.String("group", group),
			slog.String("err", err.Error()),
		)
		return
	}

	// update ownership of existing file if appropriate
	return fm.chown()
}

func checkViablePerm(perm os.FileMode) (err error) {
	if (perm & os.ModePerm) == 0 {
		err = fmt.Errorf("disabling all access permissions is not recommended")
	}

	return
}

func (fm *ManagedFile) checkPerm() (err error) {
	if err = checkViablePerm(fm.perm); err != nil {
		err = fm.err_with_err("failed viable permissions check", err)
	}

	return
}

func (fm *ManagedFile) chmod() (err error) {
	if fm.file == nil {
		return
	}

	if err = fm.checkPath(); err != nil {
		return
	}

	if err = fm.file.Chmod(fm.perm); err != nil {
		slog.Debug(
			"failed to chmod file to new permissions",
			slog.String("path", fm.path),
			slog.String("perm", fm.perm.String()),
			slog.String("err", err.Error()),
		)
		err = fmt.Errorf(
			"failed to chmod(%q, %s): %w",
			fm.path,
			fm.perm.String(),
			err,
		)
	}

	return
}

func (fm *ManagedFile) Perm() os.FileMode {
	return fm.perm
}

func (fm *ManagedFile) SetPerm(perm os.FileMode) (err error) {
	if err = checkViablePerm(perm); err != nil {
		return err
	}

	fm.perm = perm

	return fm.chmod()
}

func (fm *ManagedFile) managedFileOperationFailed(op string, err error) error {
	slog.Debug(
		"managed file operation failed",
		slog.String("operation", op),
		slog.String("path", fm.path),
		slog.String("err", err.Error()),
	)

	return err
}

func (fm *ManagedFile) Lock() (err error) {
	if err = fm.checkPath(); err != nil {
		return fm.managedFileOperationFailed("lock", err)
	}

	if err = fm.syscall.Flock(int(fm.file.Fd()), syscall.LOCK_EX); err != nil {
		return fm.dbg_with_err("failed to lock file", err)
	}

	fm.locked = true

	return
}

func (fm *ManagedFile) Unlock() (err error) {
	if err = fm.checkPath(); err != nil {
		return fm.managedFileOperationFailed("unlock", err)
	}

	if err = fm.syscall.Flock(int(fm.file.Fd()), syscall.LOCK_UN); err != nil {
		return fm.dbg_with_err("failed to lock file", err)
	}

	fm.locked = false

	return
}

func (fm *ManagedFile) IsLocked() bool {
	return fm.locked
}

func (fm *ManagedFile) openCreate(path string, perm os.FileMode) (file *os.File, err error) {
	var created bool

	// in general we expect the file to already exist, so first try to open
	// it for read + write access without creating it.
	file, err = fm.os.OpenFile(
		path,
		os.O_RDWR,
		perm,
	)

	// if the file doesn't already exist, attempt to create it
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		fm.dbg_with_err("file doesn't exist, creating it", err)

		file, err = fm.os.OpenFile(
			path,
			os.O_CREATE|os.O_RDWR,
			perm,
		)

		// we created it if there was no failure
		if err == nil {
			created = true
		}
	}

	// if we created the file, set it's ownership
	if created {

		if err = fm.chownFile(file); err != nil {
			fm.err_with_err("failed to set ownership for created file", err)
		}
	}

	if err != nil && file != nil {
		// on error ensure we close any file that we created
		file.Close()
		file = nil
	}

	return
}

func (fm *ManagedFile) Create() (err error) {
	if err = fm.checkPath(); err != nil {
		return fm.managedFileOperationFailed("create", err)
	}

	// ensure that we create files that will be accessible
	if err = fm.checkPerm(); err != nil {
		return
	}

	// if there is an existing file close it, will be reopened below
	if fm.file != nil {
		close_err := fm.Close()
		fm.dbg_with_err("failed to close existing file during create", close_err)
	}

	// attempt to open the file, creating it if needed
	fm.file, err = fm.openCreate(
		fm.path,
		fm.perm,
	)

	// fail if we couldn't open the file successfully
	if err != nil {
		return fm.err_with_err("failed to create and open file", err)
	}

	return
}

func (fm *ManagedFile) backupFileName() string {
	return fmt.Sprintf("%s.bak", fm.path)
}

func (fm *ManagedFile) Backup() (err error) {
	if err = fm.checkPath(); err != nil {
		return fm.managedFileOperationFailed("backup", err)
	}

	exists, err := fm.Exists()
	if err != nil {
		return fm.err_with_err(
			"failed to backup file",
			fmt.Errorf("unable to determine if file exists"),
		)
	}
	if !exists {
		return fm.dbg_with_err(
			"failed to backup file",
			fmt.Errorf("file doesn't exist yet"),
		)
	}

	bkupName := fm.backupFileName()
	bkup, err := fm.openCreate(bkupName, fm.perm.Perm())
	if err != nil {
		return fm.err_with_err("failed to create backup", err)
	}
	defer bkup.Close()

	// file should be locked for the duration of the backup
	if !fm.IsLocked() {
		if err := fm.Lock(); err != nil {
			return fm.dbg_with_err("failed to lock file for backup", err)
		}
		defer fm.Unlock()
	}

	// ensure we are at start of file
	if _, err = fm.file.Seek(0, 0); err != nil {
		return fm.err_with_err("failed to seek to file start", err)
	}

	// ensure we truncate the backup file
	if err = bkup.Truncate(0); err != nil {
		return fm.err_with_err("failed to backup file", err)
	}

	// ensure we are at start of backup file
	if _, err = bkup.Seek(0, 0); err != nil {
		return fm.err_with_err("failed to seek to backup file start", err)
	}

	// backup file contents
	length, err := io.Copy(bkup, fm.file)
	if err != nil {
		return fm.err_with_err("failed to backup file", err)
	}

	slog.Debug(
		"backed up file",
		slog.String("path", fm.path),
		slog.String("backup", bkupName),
		slog.Int64("length", length),
	)

	return
}

func (fm *ManagedFile) Update(updatedContent []byte) (err error) {
	if err = fm.checkPath(); err != nil {
		return fm.managedFileOperationFailed("update", err)
	}

	// file should be locked for the duration of the read
	if !fm.IsLocked() {
		if err := fm.Lock(); err != nil {
			return fm.dbg_with_err("failed to lock file for update", err)
		}
		defer fm.Unlock()
	}

	// truncate the existing file content
	if err = fm.file.Truncate(0); err != nil {
		return fm.err_with_err("failed to truncate file", err)
	}

	// ensure we are at start of file
	if _, err = fm.file.Seek(0, 0); err != nil {
		return fm.err_with_err("failed to seek to file start", err)
	}

	// write the updated content
	if _, err = fm.file.Write(updatedContent); err != nil {
		return fm.err_with_err("failed to update file", err)
	}

	// ensure written data is flushed to disk
	if err = fm.file.Sync(); err != nil {
		return fm.err_with_err("failed to flush updates to disk", err)
	}

	return
}

func (fm *ManagedFile) Read() (content []byte, err error) {
	if err = fm.checkPath(); err != nil {
		return nil, fm.managedFileOperationFailed("read", err)
	}

	// file should be locked for the duration of the read
	if !fm.IsLocked() {
		if err := fm.Lock(); err != nil {
			return nil, fm.dbg_with_err("failed to lock file for read", err)
		}
		defer fm.Unlock()
	}

	// ensure we are at start of file
	if _, err = fm.file.Seek(0, 0); err != nil {
		return nil, fm.err_with_err("failed to seek to file start", err)
	}

	return io.ReadAll(fm.file)
}

func (fm *ManagedFile) Delete() (err error) {
	if err = fm.checkPath(); err != nil {
		return fm.managedFileOperationFailed("delete", err)
	}

	if err = fm.Close(); err != nil {
		return fm.managedFileOperationFailed("delete", err)
	}

	if err = fm.os.Remove(fm.path); err != nil {
		return fmt.Errorf(
			"failed to remove file %q: %w",
			fm.path,
			fm.dbg_with_err("failed to remove file", err),
		)
	}

	return
}

func (fm *ManagedFile) Close() (err error) {
	if err = fm.checkPath(); err != nil {
		return fm.managedFileOperationFailed("close", err)
	}

	if fm.locked {
		return fm.Unlock()
	}

	if fm.file != nil {
		if err = fm.file.Close(); err != nil {
			return fmt.Errorf(
				"failed to close file %q: %w",
				fm.path,
				fm.dbg_with_err("failed to close file", err),
			)
		}
		fm.file = nil
	}

	return
}

var _ FileManager = (*ManagedFile)(nil)
