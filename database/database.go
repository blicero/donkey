// /home/krylon/go/src/github.com/blicero/donkey/database/database.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-24 17:05:45 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/blicero/donkey/common"
	"github.com/blicero/donkey/database/query"
	"github.com/blicero/donkey/logdomain"
	"github.com/blicero/donkey/model"
	"github.com/blicero/donkey/model/recordtype"
	"github.com/blicero/krylib"
	_ "github.com/mattn/go-sqlite3" // Import the database driver
)

var (
	openLock sync.Mutex
	idCnt    int64
)

// ErrTxInProgress indicates that an attempt to initiate a transaction failed
// because there is already one in progress.
var ErrTxInProgress = errors.New("A Transaction is already in progress")

// ErrNoTxInProgress indicates that an attempt was made to finish a
// transaction when none was active.
var ErrNoTxInProgress = errors.New("There is no transaction in progress")

// ErrEmptyUpdate indicates that an update operation would not change any
// values.
var ErrEmptyUpdate = errors.New("Update operation does not change any values")

// ErrInvalidValue indicates that one or more parameters passed to a method
// had values that are invalid for that operation.
var ErrInvalidValue = errors.New("Invalid value for parameter")

// ErrObjectNotFound indicates that an Object was not found in the database.
var ErrObjectNotFound = errors.New("object was not found in database")

// ErrInvalidSavepoint is returned when a user of the Database uses an unkown
// (or expired) savepoint name.
var ErrInvalidSavepoint = errors.New("that save point does not exist")

// If a query returns an error and the error text is matched by this regex, we
// consider the error as transient and try again after a short delay.
var retryPat = regexp.MustCompile("(?i)database is (?:locked|busy)")

// worthARetry returns true if an error returned from the database
// is matched by the retryPat regex.
func worthARetry(e error) bool {
	return retryPat.MatchString(e.Error())
} // func worthARetry(e error) bool

// retryDelay is the amount of time we wait before we repeat a database
// operation that failed due to a transient error.
const retryDelay = 25 * time.Millisecond

func waitForRetry() {
	time.Sleep(retryDelay)
} // func waitForRetry()

// Database is the storage backend.
//
// It is not safe to share a Database instance between goroutines, however
// opening multiple connections to the same Database is safe.
type Database struct {
	id            int64
	db            *sql.DB
	tx            *sql.Tx
	log           *log.Logger
	path          string
	spNameCounter int
	spNameCache   map[string]string
	queries       map[query.ID]*sql.Stmt
}

// Open opens a Database. If the database specified by the path does not exist,
// yet, it is created and initialized.
func Open(path string) (*Database, error) {
	var (
		err      error
		dbExists bool
		db       = &Database{
			path:          path,
			spNameCounter: 1,
			spNameCache:   make(map[string]string),
			queries:       make(map[query.ID]*sql.Stmt),
		}
	)

	openLock.Lock()
	defer openLock.Unlock()
	idCnt++
	db.id = idCnt

	if db.log, err = common.GetLogger(logdomain.Database); err != nil {
		return nil, err
	} else if common.Debug {
		db.log.Printf("[DEBUG] Open database %s\n", path)
	}

	var connstring = fmt.Sprintf("%s?_locking=NORMAL&_journal=WAL&_fk=1&recursive_triggers=0",
		path)

	if dbExists, err = krylib.Fexists(path); err != nil {
		db.log.Printf("[ERROR] Failed to check if %s already exists: %s\n",
			path,
			err.Error())
		return nil, err
	} else if db.db, err = sql.Open("sqlite3", connstring); err != nil {
		db.log.Printf("[ERROR] Failed to open %s: %s\n",
			path,
			err.Error())
		return nil, err
	}

	if !dbExists {
		if err = db.initialize(); err != nil {
			var e2 error
			if e2 = db.db.Close(); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to close database: %s\n",
					e2.Error())
				return nil, e2
			} else if e2 = os.Remove(path); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to remove database file %s: %s\n",
					db.path,
					e2.Error())
			}
			return nil, err
		}
		db.log.Printf("[INFO] Database at %s has been initialized\n",
			path)
	}

	return db, nil
} // func Open(path string) (*Database, error)

func (db *Database) initialize() error {
	var err error
	var tx *sql.Tx

	if common.Debug {
		db.log.Printf("[DEBUG] Initialize fresh database at %s\n",
			db.path)
	}

	if tx, err = db.db.Begin(); err != nil {
		db.log.Printf("[ERROR] Cannot begin transaction: %s\n",
			err.Error())
		return err
	}

	for _, q := range qInit {
		db.log.Printf("[TRACE] Execute init query:\n%s\n",
			q)
		if _, err = tx.Exec(q); err != nil {
			db.log.Printf("[ERROR] Cannot execute init query: %s\n%s\n",
				err.Error(),
				q)
			if rbErr := tx.Rollback(); rbErr != nil {
				db.log.Printf("[CANTHAPPEN] Cannot rollback transaction: %s\n",
					rbErr.Error())
				return rbErr
			}
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		db.log.Printf("[CANTHAPPEN] Failed to commit init transaction: %s\n",
			err.Error())
		return err
	}

	return nil
} // func (db *Database) initialize() error

// Close closes the database.
// If there is a pending transaction, it is rolled back.
func (db *Database) Close() error {
	// I wonder if would make more snese to panic() if something goes wrong

	var err error

	if db.tx != nil {
		if err = db.tx.Rollback(); err != nil {
			db.log.Printf("[CRITICAL] Cannot roll back pending transaction: %s\n",
				err.Error())
			return err
		}
		db.tx = nil
	}

	for key, stmt := range db.queries {
		if err = stmt.Close(); err != nil {
			db.log.Printf("[CRITICAL] Cannot close statement handle %s: %s\n",
				key,
				err.Error())
			return err
		}
		delete(db.queries, key)
	}

	if err = db.db.Close(); err != nil {
		db.log.Printf("[CRITICAL] Cannot close database: %s\n",
			err.Error())
	}

	db.db = nil
	return nil
} // func (db *Database) Close() error

func (db *Database) getQuery(id query.ID) (*sql.Stmt, error) {
	var (
		stmt  *sql.Stmt
		found bool
		err   error
	)

	if stmt, found = db.queries[id]; found {
		return stmt, nil
	} else if _, found = qDB[id]; !found {
		return nil, fmt.Errorf("Unknown Query %d",
			id)
	}

	db.log.Printf("[TRACE] Prepare query %s\n", id)

PREPARE_QUERY:
	if stmt, err = db.db.Prepare(qDB[id]); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto PREPARE_QUERY
		}

		db.log.Printf("[ERROR] Cannor parse query %s: %s\n%s\n",
			id,
			err.Error(),
			qDB[id])
		return nil, err
	}

	db.queries[id] = stmt
	return stmt, nil
} // func (db *Database) getQuery(query.ID) (*sql.Stmt, error)

func (db *Database) resetSPNamespace() {
	db.spNameCounter = 1
	db.spNameCache = make(map[string]string)
} // func (db *Database) resetSPNamespace()

func (db *Database) generateSPName(name string) string {
	var spname = fmt.Sprintf("Savepoint%05d",
		db.spNameCounter)

	db.spNameCache[name] = spname
	db.spNameCounter++
	return spname
} // func (db *Database) generateSPName() string

// PerformMaintenance performs some maintenance operations on the database.
// It cannot be called while a transaction is in progress and will block
// pretty much all access to the database while it is running.
func (db *Database) PerformMaintenance() error {
	var mQueries = []string{
		"PRAGMA wal_checkpoint(TRUNCATE)",
		"VACUUM",
		"REINDEX",
		"ANALYZE",
	}
	var err error

	if db.tx != nil {
		return ErrTxInProgress
	}

	for _, q := range mQueries {
		if _, err = db.db.Exec(q); err != nil {
			db.log.Printf("[ERROR] Failed to execute %s: %s\n",
				q,
				err.Error())
		}
	}

	return nil
} // func (db *Database) PerformMaintenance() error

// Begin begins an explicit database transaction.
// Only one transaction can be in progress at once, attempting to start one,
// while another transaction is already in progress will yield ErrTxInProgress.
func (db *Database) Begin() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Begin Transaction\n",
		db.id)

	if db.tx != nil {
		return ErrTxInProgress
	}

BEGIN_TX:
	for db.tx == nil {
		if db.tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				continue BEGIN_TX
			} else {
				db.log.Printf("[ERROR] Failed to start transaction: %s\n",
					err.Error())
				return err
			}
		}
	}

	db.resetSPNamespace()

	return nil
} // func (db *Database) Begin() error

// SavepointCreate creates a savepoint with the given name.
//
// Savepoints only make sense within a running transaction, and just like
// with explicit transactions, managing them is the responsibility of the
// user of the Database.
//
// Creating a savepoint without a surrounding transaction is not allowed,
// even though SQLite allows it.
//
// For details on how Savepoints work, check the excellent SQLite
// documentation, but here's a quick guide:
//
// Savepoints are kind-of-like transactions within a transaction: One
// can create a savepoint, make some changes to the database, and roll
// back to that savepoint, discarding all changes made between
// creating the savepoint and rolling back to it. Savepoints can be
// quite useful, but there are a few things to keep in mind:
//
//   - Savepoints exist within a transaction. When the surrounding transaction
//     is finished, all savepoints created within that transaction cease to exist,
//     no matter if the transaction is commited or rolled back.
//
//   - When the database is recovered after being interrupted during a
//     transaction, e.g. by a power outage, the entire transaction is rolled back,
//     including all savepoints that might exist.
//
//   - When a savepoint is released, nothing changes in the state of the
//     surrounding transaction. That means rolling back the surrounding
//     transaction rolls back the entire transaction, regardless of any
//     savepoints within.
//
//   - Savepoints do not nest. Releasing a savepoint releases it and *all*
//     existing savepoints that have been created before it. Rolling back to a
//     savepoint removes that savepoint and all savepoints created after it.
func (db *Database) SavepointCreate(name string) error {
	var err error

	db.log.Printf("[DEBUG] SavepointCreate(%s)\n",
		name)

	if db.tx == nil {
		return ErrNoTxInProgress
	}

SAVEPOINT:
	// It appears that the SAVEPOINT statement does not support placeholders.
	// But I do want to used named savepoints.
	// And I do want to use the given name so that no SQL injection
	// becomes possible.
	// It would be nice if the database package or at least the SQLite
	// driver offered a way to escape the string properly.
	// One possible solution would be to use names generated by the
	// Database instead of user-defined names.
	//
	// But then I need a way to use the Database-generated name
	// in rolling back and releasing the savepoint.
	// I *could* use the names strictly inside the Database, store them in
	// a map or something and hand out a key to that name to the user.
	// Since savepoint only exist within one transaction, I could even
	// re-use names from one transaction to the next.
	//
	// Ha! I could accept arbitrary names from the user, generate a
	// clean name, and store these in a map. That way the user can
	// still choose names that are outwardly visible, but they do
	// not touch the Database itself.
	//
	//if _, err = db.tx.Exec("SAVEPOINT ?", name); err != nil {
	// if _, err = db.tx.Exec("SAVEPOINT " + name); err != nil {
	// 	if worthARetry(err) {
	// 		waitForRetry()
	// 		goto SAVEPOINT
	// 	}

	// 	db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
	// 		name,
	// 		err.Error())
	// }

	var internalName = db.generateSPName(name)

	var spQuery = "SAVEPOINT " + internalName

	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
			name,
			err.Error())
	}

	return err
} // func (db *Database) SavepointCreate(name string) error

// SavepointRelease releases the Savepoint with the given name, and all
// Savepoints created before the one being release.
func (db *Database) SavepointRelease(name string) error {
	var (
		err                   error
		internalName, spQuery string
		validName             bool
	)

	db.log.Printf("[DEBUG] SavepointRelease(%s)\n",
		name)

	if db.tx != nil {
		return ErrNoTxInProgress
	}

	if internalName, validName = db.spNameCache[name]; !validName {
		db.log.Printf("[ERROR] Attempt to release unknown Savepoint %q\n",
			name)
		return ErrInvalidSavepoint
	}

	db.log.Printf("[DEBUG] Release Savepoint %q (%q)",
		name,
		db.spNameCache[name])

	spQuery = "RELEASE SAVEPOINT " + internalName

SAVEPOINT:
	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to release savepoint %s: %s\n",
			name,
			err.Error())
	} else {
		delete(db.spNameCache, internalName)
	}

	return err
} // func (db *Database) SavepointRelease(name string) error

// SavepointRollback rolls back the running transaction to the given savepoint.
func (db *Database) SavepointRollback(name string) error {
	var (
		err                   error
		internalName, spQuery string
		validName             bool
	)

	db.log.Printf("[DEBUG] SavepointRollback(%s)\n",
		name)

	if db.tx != nil {
		return ErrNoTxInProgress
	}

	if internalName, validName = db.spNameCache[name]; !validName {
		return ErrInvalidSavepoint
	}

	spQuery = "ROLLBACK TO SAVEPOINT " + internalName

SAVEPOINT:
	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
			name,
			err.Error())
	}

	delete(db.spNameCache, name)
	return err
} // func (db *Database) SavepointRollback(name string) error

// Rollback terminates a pending transaction, undoing any changes to the
// database made during that transaction.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Rollback() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Roll back Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Rollback(); err != nil {
		return fmt.Errorf("Cannot roll back database transaction: %s",
			err.Error())
	}

	db.tx = nil
	db.resetSPNamespace()

	return nil
} // func (db *Database) Rollback() error

// Commit ends the active transaction, making any changes made during that
// transaction permanent and visible to other connections.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Commit() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Commit Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Commit(); err != nil {
		return fmt.Errorf("Cannot commit transaction: %s",
			err.Error())
	}

	db.resetSPNamespace()
	db.tx = nil
	return nil
} // func (db *Database) Commit() error

// HostAdd inserts a new Host into the database.
func (db *Database) HostAdd(h *model.Host) error {
	const qid query.ID = query.HostAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
		db.log.Printf("[INFO] Start ad-hoc transaction for adding Host %s (%s)\n",
			h.Name,
			h.Addr)
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(h.Name, h.Addr, h.OS); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Host %s/%s to database: %s",
				h.Name,
				h.Addr,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var id int64

		defer rows.Close()

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("Query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			msg = fmt.Sprintf("Failed to get ID for newly added host %s/%s: %s",
				h.Name,
				h.Addr,
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return errors.New(msg)
		}

		h.ID = krylib.ID(id)
		status = true
		return nil
	}
} // func (db *Database) HostAdd(h *model.Host) error

// HostGetByID looks up a single Host by its ID.
func (db *Database) HostGetByID(id krylib.ID) (*model.Host, error) {
	const qid query.ID = query.HostGetByID
	var (
		err  error
		msg  string
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			timestamp int64
			host      = &model.Host{ID: id}
		)

		if err = rows.Scan(&host.Name, &host.Addr, &host.OS, &timestamp); err != nil {
			msg = fmt.Sprintf("Error scanning row for Host %d: %s",
				id,
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		host.LastContact = time.Unix(timestamp, 0)

		return host, nil
	}

	db.log.Printf("[DEBUG] Host %d was not found in database.\n", id)

	return nil, nil
} // func (db *Database) HostGetByID(id krylib.ID) (*model.Host, error)

// HostGetByName looks up a single Host by its name.
func (db *Database) HostGetByName(name string) (*model.Host, error) {
	const qid query.ID = query.HostGetByName
	var (
		err  error
		msg  string
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(name); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp int64
			host  = &model.Host{Name: name}
		)

		if err = rows.Scan(&host.ID, &host.Addr, &host.OS, &stamp); err != nil {
			msg = fmt.Sprintf("Error scanning row for Host %s: %s",
				name,
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		host.LastContact = time.Unix(stamp, 0)

		return host, nil
	}

	db.log.Printf("[DEBUG] Host %s was not found in database.\n", name)

	return nil, nil
} // func (db *Database) HostGetByName(name string) (*model.Host, error)

// HostGetByAddr looks up a single Host by its address.
func (db *Database) HostGetByAddr(addr string) (*model.Host, error) {
	const qid query.ID = query.HostGetByAddr
	var (
		err  error
		msg  string
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(addr); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp int64
			host  = &model.Host{Addr: addr}
		)

		if err = rows.Scan(&host.ID, &host.Name, &host.OS, &stamp); err != nil {
			msg = fmt.Sprintf("Error scanning row for Host %s: %s",
				addr,
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		host.LastContact = time.Unix(stamp, 0)

		return host, nil
	}

	db.log.Printf("[DEBUG] Host %s was not found in database.\n", addr)

	return nil, nil
} // func (db *Database) HostGetByAddr(name string) (*model.Host, error)

// HostGetAll fetches *all* Hosts from the database.
func (db *Database) HostGetAll() ([]model.Host, error) {
	const qid query.ID = query.HostGetAll
	var (
		err  error
		msg  string
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var hosts = make([]model.Host, 0, 16)

	for rows.Next() {
		var (
			stamp int64
			host  model.Host
		)

		if err = rows.Scan(&host.ID, &host.Name, &host.Addr, &host.OS, &stamp); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		host.LastContact = time.Unix(stamp, 0)
		hosts = append(hosts, host)
	}

	return hosts, nil
} // func (db *Database) HostGetAll() ([]model.Host, error)

// HostDelete deletes a single Host from the database.
func (db *Database) HostDelete(id krylib.ID) error {
	const qid query.ID = query.HostDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot delete Host %d from database: %s",
				id,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		msg = fmt.Sprintf("Failed to query query result for number of affected rows: %s",
			err.Error())
		db.log.Printf("[ERROR] %s\n", msg)
		return err
	} else if numAffected != 1 {
		db.log.Printf("[DEBUG] Failed to delete Host %d, no matching record found in database.\n",
			id)
	}

	status = true
	return nil
} // func (db *Database) HostDelete(id krylib.ID) error

// HostUpdateName updates a hosts's name.
func (db *Database) HostUpdateName(h *model.Host, name string) error {
	const qid query.ID = query.HostUpdateName
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(name, h.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot rename Host %d from %s to %s: %s",
				h.ID,
				h.Name,
				name,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		msg = fmt.Sprintf("Failed to query query result for number of affected rows: %s",
			err.Error())
		db.log.Printf("[ERROR] %s\n", msg)
		return err
	} else if numAffected != 1 {
		db.log.Printf("[DEBUG] Failed to rename Host %d (%s), no matching record found in database.\n",
			h.ID,
			h.Name)
	} else {
		h.Name = name
	}

	status = true
	return nil
} // func (db *Database) HostUpdateName(h *model.Host, name string) error

// HostUpdateAddr updates a Host's IP address in the database.
func (db *Database) HostUpdateAddr(h *model.Host, addr string) error {
	const qid query.ID = query.HostUpdateAddr
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(addr, h.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot change address of Host %d from %s to %s: %s",
				h.ID,
				h.Addr,
				addr,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		msg = fmt.Sprintf("Failed to query query result for number of affected rows: %s",
			err.Error())
		db.log.Printf("[ERROR] %s\n", msg)
		return err
	} else if numAffected != 1 {
		db.log.Printf("[DEBUG] Failed to change address of Host %d from %s to %s, no matching record found in database.\n",
			h.ID,
			h.Addr,
			addr)
	} else {
		h.Addr = addr
	}

	status = true
	return nil
} // func (db *Database) HostUpdateAddress(h *model.Host, addr string) error

// HostUpdateOS updates a Host's OS field.
func (db *Database) HostUpdateOS(h *model.Host, os string) error {
	const qid query.ID = query.HostUpdateOS
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(os, h.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot change OS of Host %d from %s to %s: %s",
				h.ID,
				h.OS,
				os,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		msg = fmt.Sprintf("Failed to query query result for number of affected rows: %s",
			err.Error())
		db.log.Printf("[ERROR] %s\n", msg)
		return err
	} else if numAffected != 1 {
		db.log.Printf("[DEBUG] Failed to change OS of Host %d from %s to %s, no matching record found in database.\n",
			h.ID,
			h.OS,
			os)
	} else {
		h.OS = os
	}

	status = true
	return nil
} // func (db *Database) HostUpdateOS(h *model.Host, os string) error

// HostUpdateLastContact updates the LastContact timestamp on a host's database record.
func (db *Database) HostUpdateLastContact(h *model.Host, stamp time.Time) error {
	const qid query.ID = query.HostUpdateLastContact
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(stamp.Unix(), h.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot change Timestamp of Host %d from %s to %s: %s",
				h.ID,
				h.LastContact.Format(common.TimestampFormat),
				stamp.Format(common.TimestampFormat),
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		msg = fmt.Sprintf("Failed to query query result for number of affected rows: %s",
			err.Error())
		db.log.Printf("[ERROR] %s\n", msg)
		return err
	} else if numAffected != 1 {
		db.log.Printf("[DEBUG] Failed to change Timestamp of Host %d from %s to %s, no matching record found in database.\n",
			h.ID,
			h.LastContact.Format(common.TimestampFormat),
			stamp.Format(common.TimestampFormat))
	} else {
		h.LastContact = stamp
	}

	status = true
	return nil
} // func (db *Database) HostUpdateLastContact(h *model.Host, stamp time.Time) error

// LoadAdd adds a system load measurement to the database.
func (db *Database) LoadAdd(l *model.Load) error {
	const qid query.ID = query.LoadAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(l.HostID, l.Timestamp.Unix(), recordtype.LoadAvg, l.Payload()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Load to database: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var id int64

		defer rows.Close()

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("Query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			msg = fmt.Sprintf("Failed to get ID for newly added Load data: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return errors.New(msg)
		}

		l.ID = krylib.ID(id)
		status = true
		return nil
	}
} // func (db *Database) LoadAdd(l *Load) error

// LoadGetByHost fetches the up to <n> most recent Load values for the given Host.
func (db *Database) LoadGetByHost(id krylib.ID, n int64) ([]model.Load, error) {
	const qid query.ID = query.LoadGetByHost
	var (
		err  error
		msg  string
		stmt *sql.Stmt
		cnt  int64
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

	if n < 0 {
		cnt = 16
	} else {
		cnt = n
	}

EXEC_QUERY:
	if rows, err = stmt.Query(id, n); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var data = make([]model.Load, 0, cnt)

	for rows.Next() {
		var (
			load  = model.Load{HostID: id}
			stamp int64
		)

		if err = rows.Scan(&load.ID, &stamp, &load.Load[0], &load.Load[1], &load.Load[2]); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		load.Timestamp = time.Unix(stamp, 0)

		data = append(data, load)
	}

	return data, nil
} // func (db *Database) LoadGetByHost(id krylib.ID) ([]model.Load, error)

// RecordAdd adds a Record to the database. Like, for real.
func (db *Database) RecordAdd(rec *model.Record) error {
	const qid query.ID = query.RecordAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
		db.log.Printf("[INFO] Start ad-hoc transaction for adding Record for Host #%d\n",
			rec.HostID)
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(rec.HostID, rec.Timestamp.Unix(), recordtype.LoadAvg, rec.Payload); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Load to database: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	var id int64

	defer rows.Close()

	if !rows.Next() {
		// CANTHAPPEN
		db.log.Printf("[ERROR] Query %s did not return a value\n",
			qid)
		return fmt.Errorf("Query %s did not return a value", qid)
	} else if err = rows.Scan(&id); err != nil {
		msg = fmt.Sprintf("Failed to get ID for newly added Load data: %s",
			err.Error())
		db.log.Printf("[ERROR] %s\n", msg)
		return errors.New(msg)
	}

	rec.ID = id
	status = true
	return nil
} // func (db *Database) RecordAdd(rec *model.Record) error

// RecordGetByHost retrieves all Records for a given Host, of all types.
// Probably a bit of a blunt instrument.
func (db *Database) RecordGetByHost(h *model.Host) ([]model.Record, error) {
	const qid query.ID = query.RecordGetByHost
	var (
		err  error
		msg  string
		stmt *sql.Stmt
	)

	if h == nil {
		return nil, krylib.ErrInvalidValue
	}

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(h.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var data = make([]model.Record, 0)

	for rows.Next() {
		var (
			rec        = model.Record{HostID: int64(h.ID)}
			stamp, src int64
		)

		if err = rows.Scan(&rec.ID, &stamp, &src, &rec.Payload); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		rec.Source = recordtype.ID(src)
		rec.Timestamp = time.Unix(stamp, 0)

		data = append(data, rec)
	}

	return data, nil
} // func (db *Database) RecordGetByHost(h *model.Host) ([]model.Record, error)

// RecordGetByType fetches all Records of a given type.
func (db *Database) RecordGetByType(id recordtype.ID) ([]model.Record, error) {
	const qid query.ID = query.RecordGetByType
	var (
		err  error
		msg  string
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var data = make([]model.Record, 0)

	for rows.Next() {
		var (
			rec   = model.Record{Source: id}
			stamp int64
		)

		if err = rows.Scan(&rec.ID, &rec.HostID, &stamp, &rec.Payload); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		rec.Timestamp = time.Unix(stamp, 0)

		data = append(data, rec)
	}

	return data, nil
} // func (db *Database) RecordGetByType(id recordtype.ID) ([]model.Record, error)

// RecordGetByHostType fetches all records for a given Host of a given type.
func (db *Database) RecordGetByHostType(h *model.Host, t recordtype.ID) ([]model.Record, error) {
	const qid query.ID = query.RecordGetByHostType
	var (
		err  error
		msg  string
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(h.ID, t); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var data = make([]model.Record, 0)

	for rows.Next() {
		var (
			rec = model.Record{
				HostID: int64(h.ID),
				Source: t,
			}
			stamp int64
		)

		if err = rows.Scan(&rec.ID, &rec.HostID, &stamp, &rec.Payload); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		rec.Timestamp = time.Unix(stamp, 0)

		data = append(data, rec)
	}

	return data, nil
} // func (db *Database) RecordGetByHostType(h *model.Host, t recordtype.ID) ([]model.Record, error)
