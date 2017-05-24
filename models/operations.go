package models

import (
	"bytes"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/vattle/sqlboiler/boil"
	"github.com/vattle/sqlboiler/queries"
	"github.com/vattle/sqlboiler/queries/qm"
	"github.com/vattle/sqlboiler/strmangle"
	"gopkg.in/nullbio/null.v6"
)

// Operation is an object representing the database table.
type Operation struct {
	ID         int64       `boil:"id" json:"id" toml:"id" yaml:"id"`
	UID        string      `boil:"uid" json:"uid" toml:"uid" yaml:"uid"`
	TypeID     int64       `boil:"type_id" json:"type_id" toml:"type_id" yaml:"type_id"`
	CreatedAt  time.Time   `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`
	Station    null.String `boil:"station" json:"station,omitempty" toml:"station" yaml:"station,omitempty"`
	UserID     null.Int64  `boil:"user_id" json:"user_id,omitempty" toml:"user_id" yaml:"user_id,omitempty"`
	Details    null.String `boil:"details" json:"details,omitempty" toml:"details" yaml:"details,omitempty"`
	Properties null.JSON   `boil:"properties" json:"properties,omitempty" toml:"properties" yaml:"properties,omitempty"`

	R *operationR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L operationL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

// operationR is where relationships are stored.
type operationR struct {
	Type  *OperationType
	User  *User
	Files FileSlice
}

// operationL is where Load methods for each relationship are stored.
type operationL struct{}

var (
	operationColumns               = []string{"id", "uid", "type_id", "created_at", "station", "user_id", "details", "properties"}
	operationColumnsWithoutDefault = []string{"uid", "type_id", "station", "user_id", "details", "properties"}
	operationColumnsWithDefault    = []string{"id", "created_at"}
	operationPrimaryKeyColumns     = []string{"id"}
)

type (
	// OperationSlice is an alias for a slice of pointers to Operation.
	// This should generally be used opposed to []Operation.
	OperationSlice []*Operation

	operationQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	operationType                 = reflect.TypeOf(&Operation{})
	operationMapping              = queries.MakeStructMapping(operationType)
	operationPrimaryKeyMapping, _ = queries.BindMapping(operationType, operationMapping, operationPrimaryKeyColumns)
	operationInsertCacheMut       sync.RWMutex
	operationInsertCache          = make(map[string]insertCache)
	operationUpdateCacheMut       sync.RWMutex
	operationUpdateCache          = make(map[string]updateCache)
	operationUpsertCacheMut       sync.RWMutex
	operationUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force bytes in case of primary key column that uses []byte (for relationship compares)
	_ = bytes.MinRead
)

// OneP returns a single operation record from the query, and panics on error.
func (q operationQuery) OneP() *Operation {
	o, err := q.One()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// One returns a single operation record from the query.
func (q operationQuery) One() (*Operation, error) {
	o := &Operation{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(o)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for operations")
	}

	return o, nil
}

// AllP returns all Operation records from the query, and panics on error.
func (q operationQuery) AllP() OperationSlice {
	o, err := q.All()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// All returns all Operation records from the query.
func (q operationQuery) All() (OperationSlice, error) {
	var o OperationSlice

	err := q.Bind(&o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to Operation slice")
	}

	return o, nil
}

// CountP returns the count of all Operation records in the query, and panics on error.
func (q operationQuery) CountP() int64 {
	c, err := q.Count()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return c
}

// Count returns the count of all Operation records in the query.
func (q operationQuery) Count() (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRow().Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count operations rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table, and panics on error.
func (q operationQuery) ExistsP() bool {
	e, err := q.Exists()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// Exists checks if the row exists in the table.
func (q operationQuery) Exists() (bool, error) {
	var count int64

	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRow().Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if operations exists")
	}

	return count > 0, nil
}

// TypeG pointed to by the foreign key.
func (o *Operation) TypeG(mods ...qm.QueryMod) operationTypeQuery {
	return o.Type(boil.GetDB(), mods...)
}

// Type pointed to by the foreign key.
func (o *Operation) Type(exec boil.Executor, mods ...qm.QueryMod) operationTypeQuery {
	queryMods := []qm.QueryMod{
		qm.Where("id=?", o.TypeID),
	}

	queryMods = append(queryMods, mods...)

	query := OperationTypes(exec, queryMods...)
	queries.SetFrom(query.Query, "\"operation_types\"")

	return query
}

// UserG pointed to by the foreign key.
func (o *Operation) UserG(mods ...qm.QueryMod) userQuery {
	return o.User(boil.GetDB(), mods...)
}

// User pointed to by the foreign key.
func (o *Operation) User(exec boil.Executor, mods ...qm.QueryMod) userQuery {
	queryMods := []qm.QueryMod{
		qm.Where("id=?", o.UserID),
	}

	queryMods = append(queryMods, mods...)

	query := Users(exec, queryMods...)
	queries.SetFrom(query.Query, "\"users\"")

	return query
}

// FilesG retrieves all the file's files.
func (o *Operation) FilesG(mods ...qm.QueryMod) fileQuery {
	return o.Files(boil.GetDB(), mods...)
}

// Files retrieves all the file's files with an executor.
func (o *Operation) Files(exec boil.Executor, mods ...qm.QueryMod) fileQuery {
	queryMods := []qm.QueryMod{
		qm.Select("\"a\".*"),
	}

	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.InnerJoin("\"files_operations\" as \"b\" on \"a\".\"id\" = \"b\".\"file_id\""),
		qm.Where("\"b\".\"operation_id\"=?", o.ID),
	)

	query := Files(exec, queryMods...)
	queries.SetFrom(query.Query, "\"files\" as \"a\"")
	return query
}

// LoadType allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (operationL) LoadType(e boil.Executor, singular bool, maybeOperation interface{}) error {
	var slice []*Operation
	var object *Operation

	count := 1
	if singular {
		object = maybeOperation.(*Operation)
	} else {
		slice = *maybeOperation.(*OperationSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &operationR{}
		}
		args[0] = object.TypeID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &operationR{}
			}
			args[i] = obj.TypeID
		}
	}

	query := fmt.Sprintf(
		"select * from \"operation_types\" where \"id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)

	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load OperationType")
	}
	defer results.Close()

	var resultSlice []*OperationType
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice OperationType")
	}

	if singular && len(resultSlice) != 0 {
		object.R.Type = resultSlice[0]
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.TypeID == foreign.ID {
				local.R.Type = foreign
				break
			}
		}
	}

	return nil
}

// LoadUser allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (operationL) LoadUser(e boil.Executor, singular bool, maybeOperation interface{}) error {
	var slice []*Operation
	var object *Operation

	count := 1
	if singular {
		object = maybeOperation.(*Operation)
	} else {
		slice = *maybeOperation.(*OperationSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &operationR{}
		}
		args[0] = object.UserID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &operationR{}
			}
			args[i] = obj.UserID
		}
	}

	query := fmt.Sprintf(
		"select * from \"users\" where \"id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)

	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load User")
	}
	defer results.Close()

	var resultSlice []*User
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice User")
	}

	if singular && len(resultSlice) != 0 {
		object.R.User = resultSlice[0]
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.UserID.Int64 == foreign.ID {
				local.R.User = foreign
				break
			}
		}
	}

	return nil
}

// LoadFiles allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (operationL) LoadFiles(e boil.Executor, singular bool, maybeOperation interface{}) error {
	var slice []*Operation
	var object *Operation

	count := 1
	if singular {
		object = maybeOperation.(*Operation)
	} else {
		slice = *maybeOperation.(*OperationSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &operationR{}
		}
		args[0] = object.ID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &operationR{}
			}
			args[i] = obj.ID
		}
	}

	query := fmt.Sprintf(
		"select \"a\".*, \"b\".\"operation_id\" from \"files\" as \"a\" inner join \"files_operations\" as \"b\" on \"a\".\"id\" = \"b\".\"file_id\" where \"b\".\"operation_id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)
	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load files")
	}
	defer results.Close()

	var resultSlice []*File

	var localJoinCols []int64
	for results.Next() {
		one := new(File)
		var localJoinCol int64

		err = results.Scan(&one.ID, &one.UID, &one.Name, &one.Size, &one.Type, &one.SubType, &one.MimeType, &one.Sha1, &one.ContentUnitID, &one.CreatedAt, &one.Language, &one.BackupCount, &one.FirstBackupTime, &one.Properties, &one.ParentID, &one.FileCreatedAt, &one.Secure, &one.Published, &localJoinCol)
		if err = results.Err(); err != nil {
			return errors.Wrap(err, "failed to plebian-bind eager loaded slice files")
		}

		resultSlice = append(resultSlice, one)
		localJoinCols = append(localJoinCols, localJoinCol)
	}

	if err = results.Err(); err != nil {
		return errors.Wrap(err, "failed to plebian-bind eager loaded slice files")
	}

	if singular {
		object.R.Files = resultSlice
		return nil
	}

	for i, foreign := range resultSlice {
		localJoinCol := localJoinCols[i]
		for _, local := range slice {
			if local.ID == localJoinCol {
				local.R.Files = append(local.R.Files, foreign)
				break
			}
		}
	}

	return nil
}

// SetTypeG of the operation to the related item.
// Sets o.R.Type to related.
// Adds o to related.R.TypeOperations.
// Uses the global database handle.
func (o *Operation) SetTypeG(insert bool, related *OperationType) error {
	return o.SetType(boil.GetDB(), insert, related)
}

// SetTypeP of the operation to the related item.
// Sets o.R.Type to related.
// Adds o to related.R.TypeOperations.
// Panics on error.
func (o *Operation) SetTypeP(exec boil.Executor, insert bool, related *OperationType) {
	if err := o.SetType(exec, insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetTypeGP of the operation to the related item.
// Sets o.R.Type to related.
// Adds o to related.R.TypeOperations.
// Uses the global database handle and panics on error.
func (o *Operation) SetTypeGP(insert bool, related *OperationType) {
	if err := o.SetType(boil.GetDB(), insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetType of the operation to the related item.
// Sets o.R.Type to related.
// Adds o to related.R.TypeOperations.
func (o *Operation) SetType(exec boil.Executor, insert bool, related *OperationType) error {
	var err error
	if insert {
		if err = related.Insert(exec); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE \"operations\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, []string{"type_id"}),
		strmangle.WhereClause("\"", "\"", 2, operationPrimaryKeyColumns),
	)
	values := []interface{}{related.ID, o.ID}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, updateQuery)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	if _, err = exec.Exec(updateQuery, values...); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	o.TypeID = related.ID

	if o.R == nil {
		o.R = &operationR{
			Type: related,
		}
	} else {
		o.R.Type = related
	}

	if related.R == nil {
		related.R = &operationTypeR{
			TypeOperations: OperationSlice{o},
		}
	} else {
		related.R.TypeOperations = append(related.R.TypeOperations, o)
	}

	return nil
}

// SetUserG of the operation to the related item.
// Sets o.R.User to related.
// Adds o to related.R.Operations.
// Uses the global database handle.
func (o *Operation) SetUserG(insert bool, related *User) error {
	return o.SetUser(boil.GetDB(), insert, related)
}

// SetUserP of the operation to the related item.
// Sets o.R.User to related.
// Adds o to related.R.Operations.
// Panics on error.
func (o *Operation) SetUserP(exec boil.Executor, insert bool, related *User) {
	if err := o.SetUser(exec, insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetUserGP of the operation to the related item.
// Sets o.R.User to related.
// Adds o to related.R.Operations.
// Uses the global database handle and panics on error.
func (o *Operation) SetUserGP(insert bool, related *User) {
	if err := o.SetUser(boil.GetDB(), insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetUser of the operation to the related item.
// Sets o.R.User to related.
// Adds o to related.R.Operations.
func (o *Operation) SetUser(exec boil.Executor, insert bool, related *User) error {
	var err error
	if insert {
		if err = related.Insert(exec); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE \"operations\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, []string{"user_id"}),
		strmangle.WhereClause("\"", "\"", 2, operationPrimaryKeyColumns),
	)
	values := []interface{}{related.ID, o.ID}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, updateQuery)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	if _, err = exec.Exec(updateQuery, values...); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	o.UserID.Int64 = related.ID
	o.UserID.Valid = true

	if o.R == nil {
		o.R = &operationR{
			User: related,
		}
	} else {
		o.R.User = related
	}

	if related.R == nil {
		related.R = &userR{
			Operations: OperationSlice{o},
		}
	} else {
		related.R.Operations = append(related.R.Operations, o)
	}

	return nil
}

// RemoveUserG relationship.
// Sets o.R.User to nil.
// Removes o from all passed in related items' relationships struct (Optional).
// Uses the global database handle.
func (o *Operation) RemoveUserG(related *User) error {
	return o.RemoveUser(boil.GetDB(), related)
}

// RemoveUserP relationship.
// Sets o.R.User to nil.
// Removes o from all passed in related items' relationships struct (Optional).
// Panics on error.
func (o *Operation) RemoveUserP(exec boil.Executor, related *User) {
	if err := o.RemoveUser(exec, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveUserGP relationship.
// Sets o.R.User to nil.
// Removes o from all passed in related items' relationships struct (Optional).
// Uses the global database handle and panics on error.
func (o *Operation) RemoveUserGP(related *User) {
	if err := o.RemoveUser(boil.GetDB(), related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveUser relationship.
// Sets o.R.User to nil.
// Removes o from all passed in related items' relationships struct (Optional).
func (o *Operation) RemoveUser(exec boil.Executor, related *User) error {
	var err error

	o.UserID.Valid = false
	if err = o.Update(exec, "user_id"); err != nil {
		o.UserID.Valid = true
		return errors.Wrap(err, "failed to update local table")
	}

	o.R.User = nil
	if related == nil || related.R == nil {
		return nil
	}

	for i, ri := range related.R.Operations {
		if o.UserID.Int64 != ri.UserID.Int64 {
			continue
		}

		ln := len(related.R.Operations)
		if ln > 1 && i < ln-1 {
			related.R.Operations[i] = related.R.Operations[ln-1]
		}
		related.R.Operations = related.R.Operations[:ln-1]
		break
	}
	return nil
}

// AddFilesG adds the given related objects to the existing relationships
// of the operation, optionally inserting them as new records.
// Appends related to o.R.Files.
// Sets related.R.Operations appropriately.
// Uses the global database handle.
func (o *Operation) AddFilesG(insert bool, related ...*File) error {
	return o.AddFiles(boil.GetDB(), insert, related...)
}

// AddFilesP adds the given related objects to the existing relationships
// of the operation, optionally inserting them as new records.
// Appends related to o.R.Files.
// Sets related.R.Operations appropriately.
// Panics on error.
func (o *Operation) AddFilesP(exec boil.Executor, insert bool, related ...*File) {
	if err := o.AddFiles(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddFilesGP adds the given related objects to the existing relationships
// of the operation, optionally inserting them as new records.
// Appends related to o.R.Files.
// Sets related.R.Operations appropriately.
// Uses the global database handle and panics on error.
func (o *Operation) AddFilesGP(insert bool, related ...*File) {
	if err := o.AddFiles(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddFiles adds the given related objects to the existing relationships
// of the operation, optionally inserting them as new records.
// Appends related to o.R.Files.
// Sets related.R.Operations appropriately.
func (o *Operation) AddFiles(exec boil.Executor, insert bool, related ...*File) error {
	var err error
	for _, rel := range related {
		if insert {
			if err = rel.Insert(exec); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		}
	}

	for _, rel := range related {
		query := "insert into \"files_operations\" (\"operation_id\", \"file_id\") values ($1, $2)"
		values := []interface{}{o.ID, rel.ID}

		if boil.DebugMode {
			fmt.Fprintln(boil.DebugWriter, query)
			fmt.Fprintln(boil.DebugWriter, values)
		}

		_, err = exec.Exec(query, values...)
		if err != nil {
			return errors.Wrap(err, "failed to insert into join table")
		}
	}
	if o.R == nil {
		o.R = &operationR{
			Files: related,
		}
	} else {
		o.R.Files = append(o.R.Files, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &fileR{
				Operations: OperationSlice{o},
			}
		} else {
			rel.R.Operations = append(rel.R.Operations, o)
		}
	}
	return nil
}

// SetFilesG removes all previously related items of the
// operation replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.Operations's Files accordingly.
// Replaces o.R.Files with related.
// Sets related.R.Operations's Files accordingly.
// Uses the global database handle.
func (o *Operation) SetFilesG(insert bool, related ...*File) error {
	return o.SetFiles(boil.GetDB(), insert, related...)
}

// SetFilesP removes all previously related items of the
// operation replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.Operations's Files accordingly.
// Replaces o.R.Files with related.
// Sets related.R.Operations's Files accordingly.
// Panics on error.
func (o *Operation) SetFilesP(exec boil.Executor, insert bool, related ...*File) {
	if err := o.SetFiles(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetFilesGP removes all previously related items of the
// operation replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.Operations's Files accordingly.
// Replaces o.R.Files with related.
// Sets related.R.Operations's Files accordingly.
// Uses the global database handle and panics on error.
func (o *Operation) SetFilesGP(insert bool, related ...*File) {
	if err := o.SetFiles(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetFiles removes all previously related items of the
// operation replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.Operations's Files accordingly.
// Replaces o.R.Files with related.
// Sets related.R.Operations's Files accordingly.
func (o *Operation) SetFiles(exec boil.Executor, insert bool, related ...*File) error {
	query := "delete from \"files_operations\" where \"operation_id\" = $1"
	values := []interface{}{o.ID}
	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err := exec.Exec(query, values...)
	if err != nil {
		return errors.Wrap(err, "failed to remove relationships before set")
	}

	removeFilesFromOperationsSlice(o, related)
	o.R.Files = nil
	return o.AddFiles(exec, insert, related...)
}

// RemoveFilesG relationships from objects passed in.
// Removes related items from R.Files (uses pointer comparison, removal does not keep order)
// Sets related.R.Operations.
// Uses the global database handle.
func (o *Operation) RemoveFilesG(related ...*File) error {
	return o.RemoveFiles(boil.GetDB(), related...)
}

// RemoveFilesP relationships from objects passed in.
// Removes related items from R.Files (uses pointer comparison, removal does not keep order)
// Sets related.R.Operations.
// Panics on error.
func (o *Operation) RemoveFilesP(exec boil.Executor, related ...*File) {
	if err := o.RemoveFiles(exec, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveFilesGP relationships from objects passed in.
// Removes related items from R.Files (uses pointer comparison, removal does not keep order)
// Sets related.R.Operations.
// Uses the global database handle and panics on error.
func (o *Operation) RemoveFilesGP(related ...*File) {
	if err := o.RemoveFiles(boil.GetDB(), related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveFiles relationships from objects passed in.
// Removes related items from R.Files (uses pointer comparison, removal does not keep order)
// Sets related.R.Operations.
func (o *Operation) RemoveFiles(exec boil.Executor, related ...*File) error {
	var err error
	query := fmt.Sprintf(
		"delete from \"files_operations\" where \"operation_id\" = $1 and \"file_id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, len(related), 1, 1),
	)
	values := []interface{}{o.ID}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err = exec.Exec(query, values...)
	if err != nil {
		return errors.Wrap(err, "failed to remove relationships before set")
	}
	removeFilesFromOperationsSlice(o, related)
	if o.R == nil {
		return nil
	}

	for _, rel := range related {
		for i, ri := range o.R.Files {
			if rel != ri {
				continue
			}

			ln := len(o.R.Files)
			if ln > 1 && i < ln-1 {
				o.R.Files[i] = o.R.Files[ln-1]
			}
			o.R.Files = o.R.Files[:ln-1]
			break
		}
	}

	return nil
}

func removeFilesFromOperationsSlice(o *Operation, related []*File) {
	for _, rel := range related {
		if rel.R == nil {
			continue
		}
		for i, ri := range rel.R.Operations {
			if o.ID != ri.ID {
				continue
			}

			ln := len(rel.R.Operations)
			if ln > 1 && i < ln-1 {
				rel.R.Operations[i] = rel.R.Operations[ln-1]
			}
			rel.R.Operations = rel.R.Operations[:ln-1]
			break
		}
	}
}

// OperationsG retrieves all records.
func OperationsG(mods ...qm.QueryMod) operationQuery {
	return Operations(boil.GetDB(), mods...)
}

// Operations retrieves all the records using an executor.
func Operations(exec boil.Executor, mods ...qm.QueryMod) operationQuery {
	mods = append(mods, qm.From("\"operations\""))
	return operationQuery{NewQuery(exec, mods...)}
}

// FindOperationG retrieves a single record by ID.
func FindOperationG(id int64, selectCols ...string) (*Operation, error) {
	return FindOperation(boil.GetDB(), id, selectCols...)
}

// FindOperationGP retrieves a single record by ID, and panics on error.
func FindOperationGP(id int64, selectCols ...string) *Operation {
	retobj, err := FindOperation(boil.GetDB(), id, selectCols...)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return retobj
}

// FindOperation retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindOperation(exec boil.Executor, id int64, selectCols ...string) (*Operation, error) {
	operationObj := &Operation{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"operations\" where \"id\"=$1", sel,
	)

	q := queries.Raw(exec, query, id)

	err := q.Bind(operationObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from operations")
	}

	return operationObj, nil
}

// FindOperationP retrieves a single record by ID with an executor, and panics on error.
func FindOperationP(exec boil.Executor, id int64, selectCols ...string) *Operation {
	retobj, err := FindOperation(exec, id, selectCols...)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return retobj
}

// InsertG a single record. See Insert for whitelist behavior description.
func (o *Operation) InsertG(whitelist ...string) error {
	return o.Insert(boil.GetDB(), whitelist...)
}

// InsertGP a single record, and panics on error. See Insert for whitelist
// behavior description.
func (o *Operation) InsertGP(whitelist ...string) {
	if err := o.Insert(boil.GetDB(), whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// InsertP a single record using an executor, and panics on error. See Insert
// for whitelist behavior description.
func (o *Operation) InsertP(exec boil.Executor, whitelist ...string) {
	if err := o.Insert(exec, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Insert a single record using an executor.
// Whitelist behavior: If a whitelist is provided, only those columns supplied are inserted
// No whitelist behavior: Without a whitelist, columns are inferred by the following rules:
// - All columns without a default value are included (i.e. name, age)
// - All columns with a default, but non-zero are included (i.e. health = 75)
func (o *Operation) Insert(exec boil.Executor, whitelist ...string) error {
	if o == nil {
		return errors.New("models: no operations provided for insertion")
	}

	var err error

	nzDefaults := queries.NonZeroDefaultSet(operationColumnsWithDefault, o)

	key := makeCacheKey(whitelist, nzDefaults)
	operationInsertCacheMut.RLock()
	cache, cached := operationInsertCache[key]
	operationInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := strmangle.InsertColumnSet(
			operationColumns,
			operationColumnsWithDefault,
			operationColumnsWithoutDefault,
			nzDefaults,
			whitelist,
		)

		cache.valueMapping, err = queries.BindMapping(operationType, operationMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(operationType, operationMapping, returnColumns)
		if err != nil {
			return err
		}
		cache.query = fmt.Sprintf("INSERT INTO \"operations\" (\"%s\") VALUES (%s)", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.IndexPlaceholders, len(wl), 1, 1))

		if len(cache.retMapping) != 0 {
			cache.query += fmt.Sprintf(" RETURNING \"%s\"", strings.Join(returnColumns, "\",\""))
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRow(cache.query, vals...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	} else {
		_, err = exec.Exec(cache.query, vals...)
	}

	if err != nil {
		return errors.Wrap(err, "models: unable to insert into operations")
	}

	if !cached {
		operationInsertCacheMut.Lock()
		operationInsertCache[key] = cache
		operationInsertCacheMut.Unlock()
	}

	return nil
}

// UpdateG a single Operation record. See Update for
// whitelist behavior description.
func (o *Operation) UpdateG(whitelist ...string) error {
	return o.Update(boil.GetDB(), whitelist...)
}

// UpdateGP a single Operation record.
// UpdateGP takes a whitelist of column names that should be updated.
// Panics on error. See Update for whitelist behavior description.
func (o *Operation) UpdateGP(whitelist ...string) {
	if err := o.Update(boil.GetDB(), whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateP uses an executor to update the Operation, and panics on error.
// See Update for whitelist behavior description.
func (o *Operation) UpdateP(exec boil.Executor, whitelist ...string) {
	err := o.Update(exec, whitelist...)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// Update uses an executor to update the Operation.
// Whitelist behavior: If a whitelist is provided, only the columns given are updated.
// No whitelist behavior: Without a whitelist, columns are inferred by the following rules:
// - All columns are inferred to start with
// - All primary keys are subtracted from this set
// Update does not automatically update the record in case of default values. Use .Reload()
// to refresh the records.
func (o *Operation) Update(exec boil.Executor, whitelist ...string) error {
	var err error
	key := makeCacheKey(whitelist, nil)
	operationUpdateCacheMut.RLock()
	cache, cached := operationUpdateCache[key]
	operationUpdateCacheMut.RUnlock()

	if !cached {
		wl := strmangle.UpdateColumnSet(operationColumns, operationPrimaryKeyColumns, whitelist)
		if len(wl) == 0 {
			return errors.New("models: unable to update operations, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"operations\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, operationPrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(operationType, operationMapping, append(wl, operationPrimaryKeyColumns...))
		if err != nil {
			return err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err = exec.Exec(cache.query, values...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update operations row")
	}

	if !cached {
		operationUpdateCacheMut.Lock()
		operationUpdateCache[key] = cache
		operationUpdateCacheMut.Unlock()
	}

	return nil
}

// UpdateAllP updates all rows with matching column names, and panics on error.
func (q operationQuery) UpdateAllP(cols M) {
	if err := q.UpdateAll(cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAll updates all rows with the specified column values.
func (q operationQuery) UpdateAll(cols M) error {
	queries.SetUpdate(q.Query, cols)

	_, err := q.Query.Exec()
	if err != nil {
		return errors.Wrap(err, "models: unable to update all for operations")
	}

	return nil
}

// UpdateAllG updates all rows with the specified column values.
func (o OperationSlice) UpdateAllG(cols M) error {
	return o.UpdateAll(boil.GetDB(), cols)
}

// UpdateAllGP updates all rows with the specified column values, and panics on error.
func (o OperationSlice) UpdateAllGP(cols M) {
	if err := o.UpdateAll(boil.GetDB(), cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAllP updates all rows with the specified column values, and panics on error.
func (o OperationSlice) UpdateAllP(exec boil.Executor, cols M) {
	if err := o.UpdateAll(exec, cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o OperationSlice) UpdateAll(exec boil.Executor, cols M) error {
	ln := int64(len(o))
	if ln == 0 {
		return nil
	}

	if len(cols) == 0 {
		return errors.New("models: update all requires at least one column argument")
	}

	colNames := make([]string, len(cols))
	args := make([]interface{}, len(cols))

	i := 0
	for name, value := range cols {
		colNames[i] = name
		args[i] = value
		i++
	}

	// Append all of the primary key values for each column
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), operationPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"UPDATE \"operations\" SET %s WHERE (\"id\") IN (%s)",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(o)*len(operationPrimaryKeyColumns), len(colNames)+1, len(operationPrimaryKeyColumns)),
	)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all in operation slice")
	}

	return nil
}

// UpsertG attempts an insert, and does an update or ignore on conflict.
func (o *Operation) UpsertG(updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) error {
	return o.Upsert(boil.GetDB(), updateOnConflict, conflictColumns, updateColumns, whitelist...)
}

// UpsertGP attempts an insert, and does an update or ignore on conflict. Panics on error.
func (o *Operation) UpsertGP(updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) {
	if err := o.Upsert(boil.GetDB(), updateOnConflict, conflictColumns, updateColumns, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpsertP attempts an insert using an executor, and does an update or ignore on conflict.
// UpsertP panics on error.
func (o *Operation) UpsertP(exec boil.Executor, updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) {
	if err := o.Upsert(exec, updateOnConflict, conflictColumns, updateColumns, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
func (o *Operation) Upsert(exec boil.Executor, updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) error {
	if o == nil {
		return errors.New("models: no operations provided for upsert")
	}

	nzDefaults := queries.NonZeroDefaultSet(operationColumnsWithDefault, o)

	// Build cache key in-line uglily - mysql vs postgres problems
	buf := strmangle.GetBuffer()
	if updateOnConflict {
		buf.WriteByte('t')
	} else {
		buf.WriteByte('f')
	}
	buf.WriteByte('.')
	for _, c := range conflictColumns {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range updateColumns {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range whitelist {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzDefaults {
		buf.WriteString(c)
	}
	key := buf.String()
	strmangle.PutBuffer(buf)

	operationUpsertCacheMut.RLock()
	cache, cached := operationUpsertCache[key]
	operationUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		var ret []string
		whitelist, ret = strmangle.InsertColumnSet(
			operationColumns,
			operationColumnsWithDefault,
			operationColumnsWithoutDefault,
			nzDefaults,
			whitelist,
		)
		update := strmangle.UpdateColumnSet(
			operationColumns,
			operationPrimaryKeyColumns,
			updateColumns,
		)
		if len(update) == 0 {
			return errors.New("models: unable to upsert operations, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(operationPrimaryKeyColumns))
			copy(conflict, operationPrimaryKeyColumns)
		}
		cache.query = queries.BuildUpsertQueryPostgres(dialect, "\"operations\"", updateOnConflict, ret, update, conflict, whitelist)

		cache.valueMapping, err = queries.BindMapping(operationType, operationMapping, whitelist)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(operationType, operationMapping, ret)
			if err != nil {
				return err
			}
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)
	var returns []interface{}
	if len(cache.retMapping) != 0 {
		returns = queries.PtrsFromMapping(value, cache.retMapping)
	}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRow(cache.query, vals...).Scan(returns...)
		if err == sql.ErrNoRows {
			err = nil // Postgres doesn't return anything when there's no update
		}
	} else {
		_, err = exec.Exec(cache.query, vals...)
	}
	if err != nil {
		return errors.Wrap(err, "models: unable to upsert operations")
	}

	if !cached {
		operationUpsertCacheMut.Lock()
		operationUpsertCache[key] = cache
		operationUpsertCacheMut.Unlock()
	}

	return nil
}

// DeleteP deletes a single Operation record with an executor.
// DeleteP will match against the primary key column to find the record to delete.
// Panics on error.
func (o *Operation) DeleteP(exec boil.Executor) {
	if err := o.Delete(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteG deletes a single Operation record.
// DeleteG will match against the primary key column to find the record to delete.
func (o *Operation) DeleteG() error {
	if o == nil {
		return errors.New("models: no Operation provided for deletion")
	}

	return o.Delete(boil.GetDB())
}

// DeleteGP deletes a single Operation record.
// DeleteGP will match against the primary key column to find the record to delete.
// Panics on error.
func (o *Operation) DeleteGP() {
	if err := o.DeleteG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Delete deletes a single Operation record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *Operation) Delete(exec boil.Executor) error {
	if o == nil {
		return errors.New("models: no Operation provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), operationPrimaryKeyMapping)
	sql := "DELETE FROM \"operations\" WHERE \"id\"=$1"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete from operations")
	}

	return nil
}

// DeleteAllP deletes all rows, and panics on error.
func (q operationQuery) DeleteAllP() {
	if err := q.DeleteAll(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAll deletes all matching rows.
func (q operationQuery) DeleteAll() error {
	if q.Query == nil {
		return errors.New("models: no operationQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	_, err := q.Query.Exec()
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from operations")
	}

	return nil
}

// DeleteAllGP deletes all rows in the slice, and panics on error.
func (o OperationSlice) DeleteAllGP() {
	if err := o.DeleteAllG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAllG deletes all rows in the slice.
func (o OperationSlice) DeleteAllG() error {
	if o == nil {
		return errors.New("models: no Operation slice provided for delete all")
	}
	return o.DeleteAll(boil.GetDB())
}

// DeleteAllP deletes all rows in the slice, using an executor, and panics on error.
func (o OperationSlice) DeleteAllP(exec boil.Executor) {
	if err := o.DeleteAll(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o OperationSlice) DeleteAll(exec boil.Executor) error {
	if o == nil {
		return errors.New("models: no Operation slice provided for delete all")
	}

	if len(o) == 0 {
		return nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), operationPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"DELETE FROM \"operations\" WHERE (%s) IN (%s)",
		strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, operationPrimaryKeyColumns), ","),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(o)*len(operationPrimaryKeyColumns), 1, len(operationPrimaryKeyColumns)),
	)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from operation slice")
	}

	return nil
}

// ReloadGP refetches the object from the database and panics on error.
func (o *Operation) ReloadGP() {
	if err := o.ReloadG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadP refetches the object from the database with an executor. Panics on error.
func (o *Operation) ReloadP(exec boil.Executor) {
	if err := o.Reload(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadG refetches the object from the database using the primary keys.
func (o *Operation) ReloadG() error {
	if o == nil {
		return errors.New("models: no Operation provided for reload")
	}

	return o.Reload(boil.GetDB())
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *Operation) Reload(exec boil.Executor) error {
	ret, err := FindOperation(exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAllGP refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
// Panics on error.
func (o *OperationSlice) ReloadAllGP() {
	if err := o.ReloadAllG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadAllP refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
// Panics on error.
func (o *OperationSlice) ReloadAllP(exec boil.Executor) {
	if err := o.ReloadAll(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadAllG refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *OperationSlice) ReloadAllG() error {
	if o == nil {
		return errors.New("models: empty OperationSlice provided for reload all")
	}

	return o.ReloadAll(boil.GetDB())
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *OperationSlice) ReloadAll(exec boil.Executor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	operations := OperationSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), operationPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"SELECT \"operations\".* FROM \"operations\" WHERE (%s) IN (%s)",
		strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, operationPrimaryKeyColumns), ","),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(*o)*len(operationPrimaryKeyColumns), 1, len(operationPrimaryKeyColumns)),
	)

	q := queries.Raw(exec, sql, args...)

	err := q.Bind(&operations)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in OperationSlice")
	}

	*o = operations

	return nil
}

// OperationExists checks if the Operation row exists.
func OperationExists(exec boil.Executor, id int64) (bool, error) {
	var exists bool

	sql := "select exists(select 1 from \"operations\" where \"id\"=$1 limit 1)"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, id)
	}

	row := exec.QueryRow(sql, id)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if operations exists")
	}

	return exists, nil
}

// OperationExistsG checks if the Operation row exists.
func OperationExistsG(id int64) (bool, error) {
	return OperationExists(boil.GetDB(), id)
}

// OperationExistsGP checks if the Operation row exists. Panics on error.
func OperationExistsGP(id int64) bool {
	e, err := OperationExists(boil.GetDB(), id)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// OperationExistsP checks if the Operation row exists. Panics on error.
func OperationExistsP(exec boil.Executor, id int64) bool {
	e, err := OperationExists(exec, id)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}
