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

// User is an object representing the database table.
type User struct {
	ID        int64       `boil:"id" json:"id" toml:"id" yaml:"id"`
	Email     string      `boil:"email" json:"email" toml:"email" yaml:"email"`
	Name      null.String `boil:"name" json:"name,omitempty" toml:"name" yaml:"name,omitempty"`
	Phone     null.String `boil:"phone" json:"phone,omitempty" toml:"phone" yaml:"phone,omitempty"`
	Comments  null.String `boil:"comments" json:"comments,omitempty" toml:"comments" yaml:"comments,omitempty"`
	CreatedAt time.Time   `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`
	UpdatedAt null.Time   `boil:"updated_at" json:"updated_at,omitempty" toml:"updated_at" yaml:"updated_at,omitempty"`
	DeletedAt null.Time   `boil:"deleted_at" json:"deleted_at,omitempty" toml:"deleted_at" yaml:"deleted_at,omitempty"`

	R *userR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L userL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

// userR is where relationships are stored.
type userR struct {
	ContentUnitI18ns ContentUnitI18nSlice
	Operations       OperationSlice
	PersonI18ns      PersonI18nSlice
	TagI18ns         TagI18nSlice
	CollectionI18ns  CollectionI18nSlice
}

// userL is where Load methods for each relationship are stored.
type userL struct{}

var (
	userColumns               = []string{"id", "email", "name", "phone", "comments", "created_at", "updated_at", "deleted_at"}
	userColumnsWithoutDefault = []string{"email", "name", "phone", "comments", "updated_at", "deleted_at"}
	userColumnsWithDefault    = []string{"id", "created_at"}
	userPrimaryKeyColumns     = []string{"id"}
)

type (
	// UserSlice is an alias for a slice of pointers to User.
	// This should generally be used opposed to []User.
	UserSlice []*User

	userQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	userType                 = reflect.TypeOf(&User{})
	userMapping              = queries.MakeStructMapping(userType)
	userPrimaryKeyMapping, _ = queries.BindMapping(userType, userMapping, userPrimaryKeyColumns)
	userInsertCacheMut       sync.RWMutex
	userInsertCache          = make(map[string]insertCache)
	userUpdateCacheMut       sync.RWMutex
	userUpdateCache          = make(map[string]updateCache)
	userUpsertCacheMut       sync.RWMutex
	userUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force bytes in case of primary key column that uses []byte (for relationship compares)
	_ = bytes.MinRead
)

// OneP returns a single user record from the query, and panics on error.
func (q userQuery) OneP() *User {
	o, err := q.One()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// One returns a single user record from the query.
func (q userQuery) One() (*User, error) {
	o := &User{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(o)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for users")
	}

	return o, nil
}

// AllP returns all User records from the query, and panics on error.
func (q userQuery) AllP() UserSlice {
	o, err := q.All()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// All returns all User records from the query.
func (q userQuery) All() (UserSlice, error) {
	var o UserSlice

	err := q.Bind(&o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to User slice")
	}

	return o, nil
}

// CountP returns the count of all User records in the query, and panics on error.
func (q userQuery) CountP() int64 {
	c, err := q.Count()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return c
}

// Count returns the count of all User records in the query.
func (q userQuery) Count() (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRow().Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count users rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table, and panics on error.
func (q userQuery) ExistsP() bool {
	e, err := q.Exists()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// Exists checks if the row exists in the table.
func (q userQuery) Exists() (bool, error) {
	var count int64

	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRow().Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if users exists")
	}

	return count > 0, nil
}

// ContentUnitI18nsG retrieves all the content_unit_i18n's content unit i18n.
func (o *User) ContentUnitI18nsG(mods ...qm.QueryMod) contentUnitI18nQuery {
	return o.ContentUnitI18ns(boil.GetDB(), mods...)
}

// ContentUnitI18ns retrieves all the content_unit_i18n's content unit i18n with an executor.
func (o *User) ContentUnitI18ns(exec boil.Executor, mods ...qm.QueryMod) contentUnitI18nQuery {
	queryMods := []qm.QueryMod{
		qm.Select("\"a\".*"),
	}

	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.Where("\"a\".\"user_id\"=?", o.ID),
	)

	query := ContentUnitI18ns(exec, queryMods...)
	queries.SetFrom(query.Query, "\"content_unit_i18n\" as \"a\"")
	return query
}

// OperationsG retrieves all the operation's operations.
func (o *User) OperationsG(mods ...qm.QueryMod) operationQuery {
	return o.Operations(boil.GetDB(), mods...)
}

// Operations retrieves all the operation's operations with an executor.
func (o *User) Operations(exec boil.Executor, mods ...qm.QueryMod) operationQuery {
	queryMods := []qm.QueryMod{
		qm.Select("\"a\".*"),
	}

	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.Where("\"a\".\"user_id\"=?", o.ID),
	)

	query := Operations(exec, queryMods...)
	queries.SetFrom(query.Query, "\"operations\" as \"a\"")
	return query
}

// PersonI18nsG retrieves all the person_i18n's person i18n.
func (o *User) PersonI18nsG(mods ...qm.QueryMod) personI18nQuery {
	return o.PersonI18ns(boil.GetDB(), mods...)
}

// PersonI18ns retrieves all the person_i18n's person i18n with an executor.
func (o *User) PersonI18ns(exec boil.Executor, mods ...qm.QueryMod) personI18nQuery {
	queryMods := []qm.QueryMod{
		qm.Select("\"a\".*"),
	}

	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.Where("\"a\".\"user_id\"=?", o.ID),
	)

	query := PersonI18ns(exec, queryMods...)
	queries.SetFrom(query.Query, "\"person_i18n\" as \"a\"")
	return query
}

// TagI18nsG retrieves all the tag_i18n's tag i18n.
func (o *User) TagI18nsG(mods ...qm.QueryMod) tagI18nQuery {
	return o.TagI18ns(boil.GetDB(), mods...)
}

// TagI18ns retrieves all the tag_i18n's tag i18n with an executor.
func (o *User) TagI18ns(exec boil.Executor, mods ...qm.QueryMod) tagI18nQuery {
	queryMods := []qm.QueryMod{
		qm.Select("\"a\".*"),
	}

	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.Where("\"a\".\"user_id\"=?", o.ID),
	)

	query := TagI18ns(exec, queryMods...)
	queries.SetFrom(query.Query, "\"tag_i18n\" as \"a\"")
	return query
}

// CollectionI18nsG retrieves all the collection_i18n's collection i18n.
func (o *User) CollectionI18nsG(mods ...qm.QueryMod) collectionI18nQuery {
	return o.CollectionI18ns(boil.GetDB(), mods...)
}

// CollectionI18ns retrieves all the collection_i18n's collection i18n with an executor.
func (o *User) CollectionI18ns(exec boil.Executor, mods ...qm.QueryMod) collectionI18nQuery {
	queryMods := []qm.QueryMod{
		qm.Select("\"a\".*"),
	}

	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.Where("\"a\".\"user_id\"=?", o.ID),
	)

	query := CollectionI18ns(exec, queryMods...)
	queries.SetFrom(query.Query, "\"collection_i18n\" as \"a\"")
	return query
}

// LoadContentUnitI18ns allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (userL) LoadContentUnitI18ns(e boil.Executor, singular bool, maybeUser interface{}) error {
	var slice []*User
	var object *User

	count := 1
	if singular {
		object = maybeUser.(*User)
	} else {
		slice = *maybeUser.(*UserSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &userR{}
		}
		args[0] = object.ID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &userR{}
			}
			args[i] = obj.ID
		}
	}

	query := fmt.Sprintf(
		"select * from \"content_unit_i18n\" where \"user_id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)
	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load content_unit_i18n")
	}
	defer results.Close()

	var resultSlice []*ContentUnitI18n
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice content_unit_i18n")
	}

	if singular {
		object.R.ContentUnitI18ns = resultSlice
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.ID == foreign.UserID.Int64 {
				local.R.ContentUnitI18ns = append(local.R.ContentUnitI18ns, foreign)
				break
			}
		}
	}

	return nil
}

// LoadOperations allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (userL) LoadOperations(e boil.Executor, singular bool, maybeUser interface{}) error {
	var slice []*User
	var object *User

	count := 1
	if singular {
		object = maybeUser.(*User)
	} else {
		slice = *maybeUser.(*UserSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &userR{}
		}
		args[0] = object.ID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &userR{}
			}
			args[i] = obj.ID
		}
	}

	query := fmt.Sprintf(
		"select * from \"operations\" where \"user_id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)
	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load operations")
	}
	defer results.Close()

	var resultSlice []*Operation
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice operations")
	}

	if singular {
		object.R.Operations = resultSlice
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.ID == foreign.UserID.Int64 {
				local.R.Operations = append(local.R.Operations, foreign)
				break
			}
		}
	}

	return nil
}

// LoadPersonI18ns allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (userL) LoadPersonI18ns(e boil.Executor, singular bool, maybeUser interface{}) error {
	var slice []*User
	var object *User

	count := 1
	if singular {
		object = maybeUser.(*User)
	} else {
		slice = *maybeUser.(*UserSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &userR{}
		}
		args[0] = object.ID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &userR{}
			}
			args[i] = obj.ID
		}
	}

	query := fmt.Sprintf(
		"select * from \"person_i18n\" where \"user_id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)
	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load person_i18n")
	}
	defer results.Close()

	var resultSlice []*PersonI18n
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice person_i18n")
	}

	if singular {
		object.R.PersonI18ns = resultSlice
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.ID == foreign.UserID.Int64 {
				local.R.PersonI18ns = append(local.R.PersonI18ns, foreign)
				break
			}
		}
	}

	return nil
}

// LoadTagI18ns allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (userL) LoadTagI18ns(e boil.Executor, singular bool, maybeUser interface{}) error {
	var slice []*User
	var object *User

	count := 1
	if singular {
		object = maybeUser.(*User)
	} else {
		slice = *maybeUser.(*UserSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &userR{}
		}
		args[0] = object.ID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &userR{}
			}
			args[i] = obj.ID
		}
	}

	query := fmt.Sprintf(
		"select * from \"tag_i18n\" where \"user_id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)
	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load tag_i18n")
	}
	defer results.Close()

	var resultSlice []*TagI18n
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice tag_i18n")
	}

	if singular {
		object.R.TagI18ns = resultSlice
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.ID == foreign.UserID.Int64 {
				local.R.TagI18ns = append(local.R.TagI18ns, foreign)
				break
			}
		}
	}

	return nil
}

// LoadCollectionI18ns allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (userL) LoadCollectionI18ns(e boil.Executor, singular bool, maybeUser interface{}) error {
	var slice []*User
	var object *User

	count := 1
	if singular {
		object = maybeUser.(*User)
	} else {
		slice = *maybeUser.(*UserSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &userR{}
		}
		args[0] = object.ID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &userR{}
			}
			args[i] = obj.ID
		}
	}

	query := fmt.Sprintf(
		"select * from \"collection_i18n\" where \"user_id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)
	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load collection_i18n")
	}
	defer results.Close()

	var resultSlice []*CollectionI18n
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice collection_i18n")
	}

	if singular {
		object.R.CollectionI18ns = resultSlice
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.ID == foreign.UserID.Int64 {
				local.R.CollectionI18ns = append(local.R.CollectionI18ns, foreign)
				break
			}
		}
	}

	return nil
}

// AddContentUnitI18nsG adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.ContentUnitI18ns.
// Sets related.R.User appropriately.
// Uses the global database handle.
func (o *User) AddContentUnitI18nsG(insert bool, related ...*ContentUnitI18n) error {
	return o.AddContentUnitI18ns(boil.GetDB(), insert, related...)
}

// AddContentUnitI18nsP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.ContentUnitI18ns.
// Sets related.R.User appropriately.
// Panics on error.
func (o *User) AddContentUnitI18nsP(exec boil.Executor, insert bool, related ...*ContentUnitI18n) {
	if err := o.AddContentUnitI18ns(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddContentUnitI18nsGP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.ContentUnitI18ns.
// Sets related.R.User appropriately.
// Uses the global database handle and panics on error.
func (o *User) AddContentUnitI18nsGP(insert bool, related ...*ContentUnitI18n) {
	if err := o.AddContentUnitI18ns(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddContentUnitI18ns adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.ContentUnitI18ns.
// Sets related.R.User appropriately.
func (o *User) AddContentUnitI18ns(exec boil.Executor, insert bool, related ...*ContentUnitI18n) error {
	var err error
	for _, rel := range related {
		if insert {
			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
			if err = rel.Insert(exec); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		} else {
			updateQuery := fmt.Sprintf(
				"UPDATE \"content_unit_i18n\" SET %s WHERE %s",
				strmangle.SetParamNames("\"", "\"", 1, []string{"user_id"}),
				strmangle.WhereClause("\"", "\"", 2, contentUnitI18nPrimaryKeyColumns),
			)
			values := []interface{}{o.ID, rel.ContentUnitID, rel.Language}

			if boil.DebugMode {
				fmt.Fprintln(boil.DebugWriter, updateQuery)
				fmt.Fprintln(boil.DebugWriter, values)
			}

			if _, err = exec.Exec(updateQuery, values...); err != nil {
				return errors.Wrap(err, "failed to update foreign table")
			}

			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
		}
	}

	if o.R == nil {
		o.R = &userR{
			ContentUnitI18ns: related,
		}
	} else {
		o.R.ContentUnitI18ns = append(o.R.ContentUnitI18ns, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &contentUnitI18nR{
				User: o,
			}
		} else {
			rel.R.User = o
		}
	}
	return nil
}

// SetContentUnitI18nsG removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's ContentUnitI18ns accordingly.
// Replaces o.R.ContentUnitI18ns with related.
// Sets related.R.User's ContentUnitI18ns accordingly.
// Uses the global database handle.
func (o *User) SetContentUnitI18nsG(insert bool, related ...*ContentUnitI18n) error {
	return o.SetContentUnitI18ns(boil.GetDB(), insert, related...)
}

// SetContentUnitI18nsP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's ContentUnitI18ns accordingly.
// Replaces o.R.ContentUnitI18ns with related.
// Sets related.R.User's ContentUnitI18ns accordingly.
// Panics on error.
func (o *User) SetContentUnitI18nsP(exec boil.Executor, insert bool, related ...*ContentUnitI18n) {
	if err := o.SetContentUnitI18ns(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetContentUnitI18nsGP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's ContentUnitI18ns accordingly.
// Replaces o.R.ContentUnitI18ns with related.
// Sets related.R.User's ContentUnitI18ns accordingly.
// Uses the global database handle and panics on error.
func (o *User) SetContentUnitI18nsGP(insert bool, related ...*ContentUnitI18n) {
	if err := o.SetContentUnitI18ns(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetContentUnitI18ns removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's ContentUnitI18ns accordingly.
// Replaces o.R.ContentUnitI18ns with related.
// Sets related.R.User's ContentUnitI18ns accordingly.
func (o *User) SetContentUnitI18ns(exec boil.Executor, insert bool, related ...*ContentUnitI18n) error {
	query := "update \"content_unit_i18n\" set \"user_id\" = null where \"user_id\" = $1"
	values := []interface{}{o.ID}
	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err := exec.Exec(query, values...)
	if err != nil {
		return errors.Wrap(err, "failed to remove relationships before set")
	}

	if o.R != nil {
		for _, rel := range o.R.ContentUnitI18ns {
			rel.UserID.Valid = false
			if rel.R == nil {
				continue
			}

			rel.R.User = nil
		}

		o.R.ContentUnitI18ns = nil
	}
	return o.AddContentUnitI18ns(exec, insert, related...)
}

// RemoveContentUnitI18nsG relationships from objects passed in.
// Removes related items from R.ContentUnitI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle.
func (o *User) RemoveContentUnitI18nsG(related ...*ContentUnitI18n) error {
	return o.RemoveContentUnitI18ns(boil.GetDB(), related...)
}

// RemoveContentUnitI18nsP relationships from objects passed in.
// Removes related items from R.ContentUnitI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Panics on error.
func (o *User) RemoveContentUnitI18nsP(exec boil.Executor, related ...*ContentUnitI18n) {
	if err := o.RemoveContentUnitI18ns(exec, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveContentUnitI18nsGP relationships from objects passed in.
// Removes related items from R.ContentUnitI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle and panics on error.
func (o *User) RemoveContentUnitI18nsGP(related ...*ContentUnitI18n) {
	if err := o.RemoveContentUnitI18ns(boil.GetDB(), related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveContentUnitI18ns relationships from objects passed in.
// Removes related items from R.ContentUnitI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
func (o *User) RemoveContentUnitI18ns(exec boil.Executor, related ...*ContentUnitI18n) error {
	var err error
	for _, rel := range related {
		rel.UserID.Valid = false
		if rel.R != nil {
			rel.R.User = nil
		}
		if err = rel.Update(exec, "user_id"); err != nil {
			return err
		}
	}
	if o.R == nil {
		return nil
	}

	for _, rel := range related {
		for i, ri := range o.R.ContentUnitI18ns {
			if rel != ri {
				continue
			}

			ln := len(o.R.ContentUnitI18ns)
			if ln > 1 && i < ln-1 {
				o.R.ContentUnitI18ns[i] = o.R.ContentUnitI18ns[ln-1]
			}
			o.R.ContentUnitI18ns = o.R.ContentUnitI18ns[:ln-1]
			break
		}
	}

	return nil
}

// AddOperationsG adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.Operations.
// Sets related.R.User appropriately.
// Uses the global database handle.
func (o *User) AddOperationsG(insert bool, related ...*Operation) error {
	return o.AddOperations(boil.GetDB(), insert, related...)
}

// AddOperationsP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.Operations.
// Sets related.R.User appropriately.
// Panics on error.
func (o *User) AddOperationsP(exec boil.Executor, insert bool, related ...*Operation) {
	if err := o.AddOperations(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddOperationsGP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.Operations.
// Sets related.R.User appropriately.
// Uses the global database handle and panics on error.
func (o *User) AddOperationsGP(insert bool, related ...*Operation) {
	if err := o.AddOperations(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddOperations adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.Operations.
// Sets related.R.User appropriately.
func (o *User) AddOperations(exec boil.Executor, insert bool, related ...*Operation) error {
	var err error
	for _, rel := range related {
		if insert {
			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
			if err = rel.Insert(exec); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		} else {
			updateQuery := fmt.Sprintf(
				"UPDATE \"operations\" SET %s WHERE %s",
				strmangle.SetParamNames("\"", "\"", 1, []string{"user_id"}),
				strmangle.WhereClause("\"", "\"", 2, operationPrimaryKeyColumns),
			)
			values := []interface{}{o.ID, rel.ID}

			if boil.DebugMode {
				fmt.Fprintln(boil.DebugWriter, updateQuery)
				fmt.Fprintln(boil.DebugWriter, values)
			}

			if _, err = exec.Exec(updateQuery, values...); err != nil {
				return errors.Wrap(err, "failed to update foreign table")
			}

			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
		}
	}

	if o.R == nil {
		o.R = &userR{
			Operations: related,
		}
	} else {
		o.R.Operations = append(o.R.Operations, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &operationR{
				User: o,
			}
		} else {
			rel.R.User = o
		}
	}
	return nil
}

// SetOperationsG removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's Operations accordingly.
// Replaces o.R.Operations with related.
// Sets related.R.User's Operations accordingly.
// Uses the global database handle.
func (o *User) SetOperationsG(insert bool, related ...*Operation) error {
	return o.SetOperations(boil.GetDB(), insert, related...)
}

// SetOperationsP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's Operations accordingly.
// Replaces o.R.Operations with related.
// Sets related.R.User's Operations accordingly.
// Panics on error.
func (o *User) SetOperationsP(exec boil.Executor, insert bool, related ...*Operation) {
	if err := o.SetOperations(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetOperationsGP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's Operations accordingly.
// Replaces o.R.Operations with related.
// Sets related.R.User's Operations accordingly.
// Uses the global database handle and panics on error.
func (o *User) SetOperationsGP(insert bool, related ...*Operation) {
	if err := o.SetOperations(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetOperations removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's Operations accordingly.
// Replaces o.R.Operations with related.
// Sets related.R.User's Operations accordingly.
func (o *User) SetOperations(exec boil.Executor, insert bool, related ...*Operation) error {
	query := "update \"operations\" set \"user_id\" = null where \"user_id\" = $1"
	values := []interface{}{o.ID}
	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err := exec.Exec(query, values...)
	if err != nil {
		return errors.Wrap(err, "failed to remove relationships before set")
	}

	if o.R != nil {
		for _, rel := range o.R.Operations {
			rel.UserID.Valid = false
			if rel.R == nil {
				continue
			}

			rel.R.User = nil
		}

		o.R.Operations = nil
	}
	return o.AddOperations(exec, insert, related...)
}

// RemoveOperationsG relationships from objects passed in.
// Removes related items from R.Operations (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle.
func (o *User) RemoveOperationsG(related ...*Operation) error {
	return o.RemoveOperations(boil.GetDB(), related...)
}

// RemoveOperationsP relationships from objects passed in.
// Removes related items from R.Operations (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Panics on error.
func (o *User) RemoveOperationsP(exec boil.Executor, related ...*Operation) {
	if err := o.RemoveOperations(exec, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveOperationsGP relationships from objects passed in.
// Removes related items from R.Operations (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle and panics on error.
func (o *User) RemoveOperationsGP(related ...*Operation) {
	if err := o.RemoveOperations(boil.GetDB(), related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveOperations relationships from objects passed in.
// Removes related items from R.Operations (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
func (o *User) RemoveOperations(exec boil.Executor, related ...*Operation) error {
	var err error
	for _, rel := range related {
		rel.UserID.Valid = false
		if rel.R != nil {
			rel.R.User = nil
		}
		if err = rel.Update(exec, "user_id"); err != nil {
			return err
		}
	}
	if o.R == nil {
		return nil
	}

	for _, rel := range related {
		for i, ri := range o.R.Operations {
			if rel != ri {
				continue
			}

			ln := len(o.R.Operations)
			if ln > 1 && i < ln-1 {
				o.R.Operations[i] = o.R.Operations[ln-1]
			}
			o.R.Operations = o.R.Operations[:ln-1]
			break
		}
	}

	return nil
}

// AddPersonI18nsG adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.PersonI18ns.
// Sets related.R.User appropriately.
// Uses the global database handle.
func (o *User) AddPersonI18nsG(insert bool, related ...*PersonI18n) error {
	return o.AddPersonI18ns(boil.GetDB(), insert, related...)
}

// AddPersonI18nsP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.PersonI18ns.
// Sets related.R.User appropriately.
// Panics on error.
func (o *User) AddPersonI18nsP(exec boil.Executor, insert bool, related ...*PersonI18n) {
	if err := o.AddPersonI18ns(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddPersonI18nsGP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.PersonI18ns.
// Sets related.R.User appropriately.
// Uses the global database handle and panics on error.
func (o *User) AddPersonI18nsGP(insert bool, related ...*PersonI18n) {
	if err := o.AddPersonI18ns(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddPersonI18ns adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.PersonI18ns.
// Sets related.R.User appropriately.
func (o *User) AddPersonI18ns(exec boil.Executor, insert bool, related ...*PersonI18n) error {
	var err error
	for _, rel := range related {
		if insert {
			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
			if err = rel.Insert(exec); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		} else {
			updateQuery := fmt.Sprintf(
				"UPDATE \"person_i18n\" SET %s WHERE %s",
				strmangle.SetParamNames("\"", "\"", 1, []string{"user_id"}),
				strmangle.WhereClause("\"", "\"", 2, personI18nPrimaryKeyColumns),
			)
			values := []interface{}{o.ID, rel.PersonID, rel.Language}

			if boil.DebugMode {
				fmt.Fprintln(boil.DebugWriter, updateQuery)
				fmt.Fprintln(boil.DebugWriter, values)
			}

			if _, err = exec.Exec(updateQuery, values...); err != nil {
				return errors.Wrap(err, "failed to update foreign table")
			}

			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
		}
	}

	if o.R == nil {
		o.R = &userR{
			PersonI18ns: related,
		}
	} else {
		o.R.PersonI18ns = append(o.R.PersonI18ns, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &personI18nR{
				User: o,
			}
		} else {
			rel.R.User = o
		}
	}
	return nil
}

// SetPersonI18nsG removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's PersonI18ns accordingly.
// Replaces o.R.PersonI18ns with related.
// Sets related.R.User's PersonI18ns accordingly.
// Uses the global database handle.
func (o *User) SetPersonI18nsG(insert bool, related ...*PersonI18n) error {
	return o.SetPersonI18ns(boil.GetDB(), insert, related...)
}

// SetPersonI18nsP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's PersonI18ns accordingly.
// Replaces o.R.PersonI18ns with related.
// Sets related.R.User's PersonI18ns accordingly.
// Panics on error.
func (o *User) SetPersonI18nsP(exec boil.Executor, insert bool, related ...*PersonI18n) {
	if err := o.SetPersonI18ns(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetPersonI18nsGP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's PersonI18ns accordingly.
// Replaces o.R.PersonI18ns with related.
// Sets related.R.User's PersonI18ns accordingly.
// Uses the global database handle and panics on error.
func (o *User) SetPersonI18nsGP(insert bool, related ...*PersonI18n) {
	if err := o.SetPersonI18ns(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetPersonI18ns removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's PersonI18ns accordingly.
// Replaces o.R.PersonI18ns with related.
// Sets related.R.User's PersonI18ns accordingly.
func (o *User) SetPersonI18ns(exec boil.Executor, insert bool, related ...*PersonI18n) error {
	query := "update \"person_i18n\" set \"user_id\" = null where \"user_id\" = $1"
	values := []interface{}{o.ID}
	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err := exec.Exec(query, values...)
	if err != nil {
		return errors.Wrap(err, "failed to remove relationships before set")
	}

	if o.R != nil {
		for _, rel := range o.R.PersonI18ns {
			rel.UserID.Valid = false
			if rel.R == nil {
				continue
			}

			rel.R.User = nil
		}

		o.R.PersonI18ns = nil
	}
	return o.AddPersonI18ns(exec, insert, related...)
}

// RemovePersonI18nsG relationships from objects passed in.
// Removes related items from R.PersonI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle.
func (o *User) RemovePersonI18nsG(related ...*PersonI18n) error {
	return o.RemovePersonI18ns(boil.GetDB(), related...)
}

// RemovePersonI18nsP relationships from objects passed in.
// Removes related items from R.PersonI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Panics on error.
func (o *User) RemovePersonI18nsP(exec boil.Executor, related ...*PersonI18n) {
	if err := o.RemovePersonI18ns(exec, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemovePersonI18nsGP relationships from objects passed in.
// Removes related items from R.PersonI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle and panics on error.
func (o *User) RemovePersonI18nsGP(related ...*PersonI18n) {
	if err := o.RemovePersonI18ns(boil.GetDB(), related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemovePersonI18ns relationships from objects passed in.
// Removes related items from R.PersonI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
func (o *User) RemovePersonI18ns(exec boil.Executor, related ...*PersonI18n) error {
	var err error
	for _, rel := range related {
		rel.UserID.Valid = false
		if rel.R != nil {
			rel.R.User = nil
		}
		if err = rel.Update(exec, "user_id"); err != nil {
			return err
		}
	}
	if o.R == nil {
		return nil
	}

	for _, rel := range related {
		for i, ri := range o.R.PersonI18ns {
			if rel != ri {
				continue
			}

			ln := len(o.R.PersonI18ns)
			if ln > 1 && i < ln-1 {
				o.R.PersonI18ns[i] = o.R.PersonI18ns[ln-1]
			}
			o.R.PersonI18ns = o.R.PersonI18ns[:ln-1]
			break
		}
	}

	return nil
}

// AddTagI18nsG adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.TagI18ns.
// Sets related.R.User appropriately.
// Uses the global database handle.
func (o *User) AddTagI18nsG(insert bool, related ...*TagI18n) error {
	return o.AddTagI18ns(boil.GetDB(), insert, related...)
}

// AddTagI18nsP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.TagI18ns.
// Sets related.R.User appropriately.
// Panics on error.
func (o *User) AddTagI18nsP(exec boil.Executor, insert bool, related ...*TagI18n) {
	if err := o.AddTagI18ns(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddTagI18nsGP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.TagI18ns.
// Sets related.R.User appropriately.
// Uses the global database handle and panics on error.
func (o *User) AddTagI18nsGP(insert bool, related ...*TagI18n) {
	if err := o.AddTagI18ns(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddTagI18ns adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.TagI18ns.
// Sets related.R.User appropriately.
func (o *User) AddTagI18ns(exec boil.Executor, insert bool, related ...*TagI18n) error {
	var err error
	for _, rel := range related {
		if insert {
			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
			if err = rel.Insert(exec); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		} else {
			updateQuery := fmt.Sprintf(
				"UPDATE \"tag_i18n\" SET %s WHERE %s",
				strmangle.SetParamNames("\"", "\"", 1, []string{"user_id"}),
				strmangle.WhereClause("\"", "\"", 2, tagI18nPrimaryKeyColumns),
			)
			values := []interface{}{o.ID, rel.TagID, rel.Language}

			if boil.DebugMode {
				fmt.Fprintln(boil.DebugWriter, updateQuery)
				fmt.Fprintln(boil.DebugWriter, values)
			}

			if _, err = exec.Exec(updateQuery, values...); err != nil {
				return errors.Wrap(err, "failed to update foreign table")
			}

			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
		}
	}

	if o.R == nil {
		o.R = &userR{
			TagI18ns: related,
		}
	} else {
		o.R.TagI18ns = append(o.R.TagI18ns, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &tagI18nR{
				User: o,
			}
		} else {
			rel.R.User = o
		}
	}
	return nil
}

// SetTagI18nsG removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's TagI18ns accordingly.
// Replaces o.R.TagI18ns with related.
// Sets related.R.User's TagI18ns accordingly.
// Uses the global database handle.
func (o *User) SetTagI18nsG(insert bool, related ...*TagI18n) error {
	return o.SetTagI18ns(boil.GetDB(), insert, related...)
}

// SetTagI18nsP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's TagI18ns accordingly.
// Replaces o.R.TagI18ns with related.
// Sets related.R.User's TagI18ns accordingly.
// Panics on error.
func (o *User) SetTagI18nsP(exec boil.Executor, insert bool, related ...*TagI18n) {
	if err := o.SetTagI18ns(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetTagI18nsGP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's TagI18ns accordingly.
// Replaces o.R.TagI18ns with related.
// Sets related.R.User's TagI18ns accordingly.
// Uses the global database handle and panics on error.
func (o *User) SetTagI18nsGP(insert bool, related ...*TagI18n) {
	if err := o.SetTagI18ns(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetTagI18ns removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's TagI18ns accordingly.
// Replaces o.R.TagI18ns with related.
// Sets related.R.User's TagI18ns accordingly.
func (o *User) SetTagI18ns(exec boil.Executor, insert bool, related ...*TagI18n) error {
	query := "update \"tag_i18n\" set \"user_id\" = null where \"user_id\" = $1"
	values := []interface{}{o.ID}
	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err := exec.Exec(query, values...)
	if err != nil {
		return errors.Wrap(err, "failed to remove relationships before set")
	}

	if o.R != nil {
		for _, rel := range o.R.TagI18ns {
			rel.UserID.Valid = false
			if rel.R == nil {
				continue
			}

			rel.R.User = nil
		}

		o.R.TagI18ns = nil
	}
	return o.AddTagI18ns(exec, insert, related...)
}

// RemoveTagI18nsG relationships from objects passed in.
// Removes related items from R.TagI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle.
func (o *User) RemoveTagI18nsG(related ...*TagI18n) error {
	return o.RemoveTagI18ns(boil.GetDB(), related...)
}

// RemoveTagI18nsP relationships from objects passed in.
// Removes related items from R.TagI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Panics on error.
func (o *User) RemoveTagI18nsP(exec boil.Executor, related ...*TagI18n) {
	if err := o.RemoveTagI18ns(exec, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveTagI18nsGP relationships from objects passed in.
// Removes related items from R.TagI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle and panics on error.
func (o *User) RemoveTagI18nsGP(related ...*TagI18n) {
	if err := o.RemoveTagI18ns(boil.GetDB(), related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveTagI18ns relationships from objects passed in.
// Removes related items from R.TagI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
func (o *User) RemoveTagI18ns(exec boil.Executor, related ...*TagI18n) error {
	var err error
	for _, rel := range related {
		rel.UserID.Valid = false
		if rel.R != nil {
			rel.R.User = nil
		}
		if err = rel.Update(exec, "user_id"); err != nil {
			return err
		}
	}
	if o.R == nil {
		return nil
	}

	for _, rel := range related {
		for i, ri := range o.R.TagI18ns {
			if rel != ri {
				continue
			}

			ln := len(o.R.TagI18ns)
			if ln > 1 && i < ln-1 {
				o.R.TagI18ns[i] = o.R.TagI18ns[ln-1]
			}
			o.R.TagI18ns = o.R.TagI18ns[:ln-1]
			break
		}
	}

	return nil
}

// AddCollectionI18nsG adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.CollectionI18ns.
// Sets related.R.User appropriately.
// Uses the global database handle.
func (o *User) AddCollectionI18nsG(insert bool, related ...*CollectionI18n) error {
	return o.AddCollectionI18ns(boil.GetDB(), insert, related...)
}

// AddCollectionI18nsP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.CollectionI18ns.
// Sets related.R.User appropriately.
// Panics on error.
func (o *User) AddCollectionI18nsP(exec boil.Executor, insert bool, related ...*CollectionI18n) {
	if err := o.AddCollectionI18ns(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddCollectionI18nsGP adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.CollectionI18ns.
// Sets related.R.User appropriately.
// Uses the global database handle and panics on error.
func (o *User) AddCollectionI18nsGP(insert bool, related ...*CollectionI18n) {
	if err := o.AddCollectionI18ns(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// AddCollectionI18ns adds the given related objects to the existing relationships
// of the user, optionally inserting them as new records.
// Appends related to o.R.CollectionI18ns.
// Sets related.R.User appropriately.
func (o *User) AddCollectionI18ns(exec boil.Executor, insert bool, related ...*CollectionI18n) error {
	var err error
	for _, rel := range related {
		if insert {
			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
			if err = rel.Insert(exec); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		} else {
			updateQuery := fmt.Sprintf(
				"UPDATE \"collection_i18n\" SET %s WHERE %s",
				strmangle.SetParamNames("\"", "\"", 1, []string{"user_id"}),
				strmangle.WhereClause("\"", "\"", 2, collectionI18nPrimaryKeyColumns),
			)
			values := []interface{}{o.ID, rel.CollectionID, rel.Language}

			if boil.DebugMode {
				fmt.Fprintln(boil.DebugWriter, updateQuery)
				fmt.Fprintln(boil.DebugWriter, values)
			}

			if _, err = exec.Exec(updateQuery, values...); err != nil {
				return errors.Wrap(err, "failed to update foreign table")
			}

			rel.UserID.Int64 = o.ID
			rel.UserID.Valid = true
		}
	}

	if o.R == nil {
		o.R = &userR{
			CollectionI18ns: related,
		}
	} else {
		o.R.CollectionI18ns = append(o.R.CollectionI18ns, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &collectionI18nR{
				User: o,
			}
		} else {
			rel.R.User = o
		}
	}
	return nil
}

// SetCollectionI18nsG removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's CollectionI18ns accordingly.
// Replaces o.R.CollectionI18ns with related.
// Sets related.R.User's CollectionI18ns accordingly.
// Uses the global database handle.
func (o *User) SetCollectionI18nsG(insert bool, related ...*CollectionI18n) error {
	return o.SetCollectionI18ns(boil.GetDB(), insert, related...)
}

// SetCollectionI18nsP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's CollectionI18ns accordingly.
// Replaces o.R.CollectionI18ns with related.
// Sets related.R.User's CollectionI18ns accordingly.
// Panics on error.
func (o *User) SetCollectionI18nsP(exec boil.Executor, insert bool, related ...*CollectionI18n) {
	if err := o.SetCollectionI18ns(exec, insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetCollectionI18nsGP removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's CollectionI18ns accordingly.
// Replaces o.R.CollectionI18ns with related.
// Sets related.R.User's CollectionI18ns accordingly.
// Uses the global database handle and panics on error.
func (o *User) SetCollectionI18nsGP(insert bool, related ...*CollectionI18n) {
	if err := o.SetCollectionI18ns(boil.GetDB(), insert, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetCollectionI18ns removes all previously related items of the
// user replacing them completely with the passed
// in related items, optionally inserting them as new records.
// Sets o.R.User's CollectionI18ns accordingly.
// Replaces o.R.CollectionI18ns with related.
// Sets related.R.User's CollectionI18ns accordingly.
func (o *User) SetCollectionI18ns(exec boil.Executor, insert bool, related ...*CollectionI18n) error {
	query := "update \"collection_i18n\" set \"user_id\" = null where \"user_id\" = $1"
	values := []interface{}{o.ID}
	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err := exec.Exec(query, values...)
	if err != nil {
		return errors.Wrap(err, "failed to remove relationships before set")
	}

	if o.R != nil {
		for _, rel := range o.R.CollectionI18ns {
			rel.UserID.Valid = false
			if rel.R == nil {
				continue
			}

			rel.R.User = nil
		}

		o.R.CollectionI18ns = nil
	}
	return o.AddCollectionI18ns(exec, insert, related...)
}

// RemoveCollectionI18nsG relationships from objects passed in.
// Removes related items from R.CollectionI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle.
func (o *User) RemoveCollectionI18nsG(related ...*CollectionI18n) error {
	return o.RemoveCollectionI18ns(boil.GetDB(), related...)
}

// RemoveCollectionI18nsP relationships from objects passed in.
// Removes related items from R.CollectionI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Panics on error.
func (o *User) RemoveCollectionI18nsP(exec boil.Executor, related ...*CollectionI18n) {
	if err := o.RemoveCollectionI18ns(exec, related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveCollectionI18nsGP relationships from objects passed in.
// Removes related items from R.CollectionI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
// Uses the global database handle and panics on error.
func (o *User) RemoveCollectionI18nsGP(related ...*CollectionI18n) {
	if err := o.RemoveCollectionI18ns(boil.GetDB(), related...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveCollectionI18ns relationships from objects passed in.
// Removes related items from R.CollectionI18ns (uses pointer comparison, removal does not keep order)
// Sets related.R.User.
func (o *User) RemoveCollectionI18ns(exec boil.Executor, related ...*CollectionI18n) error {
	var err error
	for _, rel := range related {
		rel.UserID.Valid = false
		if rel.R != nil {
			rel.R.User = nil
		}
		if err = rel.Update(exec, "user_id"); err != nil {
			return err
		}
	}
	if o.R == nil {
		return nil
	}

	for _, rel := range related {
		for i, ri := range o.R.CollectionI18ns {
			if rel != ri {
				continue
			}

			ln := len(o.R.CollectionI18ns)
			if ln > 1 && i < ln-1 {
				o.R.CollectionI18ns[i] = o.R.CollectionI18ns[ln-1]
			}
			o.R.CollectionI18ns = o.R.CollectionI18ns[:ln-1]
			break
		}
	}

	return nil
}

// UsersG retrieves all records.
func UsersG(mods ...qm.QueryMod) userQuery {
	return Users(boil.GetDB(), mods...)
}

// Users retrieves all the records using an executor.
func Users(exec boil.Executor, mods ...qm.QueryMod) userQuery {
	mods = append(mods, qm.From("\"users\""))
	return userQuery{NewQuery(exec, mods...)}
}

// FindUserG retrieves a single record by ID.
func FindUserG(id int64, selectCols ...string) (*User, error) {
	return FindUser(boil.GetDB(), id, selectCols...)
}

// FindUserGP retrieves a single record by ID, and panics on error.
func FindUserGP(id int64, selectCols ...string) *User {
	retobj, err := FindUser(boil.GetDB(), id, selectCols...)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return retobj
}

// FindUser retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindUser(exec boil.Executor, id int64, selectCols ...string) (*User, error) {
	userObj := &User{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"users\" where \"id\"=$1", sel,
	)

	q := queries.Raw(exec, query, id)

	err := q.Bind(userObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from users")
	}

	return userObj, nil
}

// FindUserP retrieves a single record by ID with an executor, and panics on error.
func FindUserP(exec boil.Executor, id int64, selectCols ...string) *User {
	retobj, err := FindUser(exec, id, selectCols...)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return retobj
}

// InsertG a single record. See Insert for whitelist behavior description.
func (o *User) InsertG(whitelist ...string) error {
	return o.Insert(boil.GetDB(), whitelist...)
}

// InsertGP a single record, and panics on error. See Insert for whitelist
// behavior description.
func (o *User) InsertGP(whitelist ...string) {
	if err := o.Insert(boil.GetDB(), whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// InsertP a single record using an executor, and panics on error. See Insert
// for whitelist behavior description.
func (o *User) InsertP(exec boil.Executor, whitelist ...string) {
	if err := o.Insert(exec, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Insert a single record using an executor.
// Whitelist behavior: If a whitelist is provided, only those columns supplied are inserted
// No whitelist behavior: Without a whitelist, columns are inferred by the following rules:
// - All columns without a default value are included (i.e. name, age)
// - All columns with a default, but non-zero are included (i.e. health = 75)
func (o *User) Insert(exec boil.Executor, whitelist ...string) error {
	if o == nil {
		return errors.New("models: no users provided for insertion")
	}

	var err error

	nzDefaults := queries.NonZeroDefaultSet(userColumnsWithDefault, o)

	key := makeCacheKey(whitelist, nzDefaults)
	userInsertCacheMut.RLock()
	cache, cached := userInsertCache[key]
	userInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := strmangle.InsertColumnSet(
			userColumns,
			userColumnsWithDefault,
			userColumnsWithoutDefault,
			nzDefaults,
			whitelist,
		)

		cache.valueMapping, err = queries.BindMapping(userType, userMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(userType, userMapping, returnColumns)
		if err != nil {
			return err
		}
		cache.query = fmt.Sprintf("INSERT INTO \"users\" (\"%s\") VALUES (%s)", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.IndexPlaceholders, len(wl), 1, 1))

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
		return errors.Wrap(err, "models: unable to insert into users")
	}

	if !cached {
		userInsertCacheMut.Lock()
		userInsertCache[key] = cache
		userInsertCacheMut.Unlock()
	}

	return nil
}

// UpdateG a single User record. See Update for
// whitelist behavior description.
func (o *User) UpdateG(whitelist ...string) error {
	return o.Update(boil.GetDB(), whitelist...)
}

// UpdateGP a single User record.
// UpdateGP takes a whitelist of column names that should be updated.
// Panics on error. See Update for whitelist behavior description.
func (o *User) UpdateGP(whitelist ...string) {
	if err := o.Update(boil.GetDB(), whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateP uses an executor to update the User, and panics on error.
// See Update for whitelist behavior description.
func (o *User) UpdateP(exec boil.Executor, whitelist ...string) {
	err := o.Update(exec, whitelist...)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// Update uses an executor to update the User.
// Whitelist behavior: If a whitelist is provided, only the columns given are updated.
// No whitelist behavior: Without a whitelist, columns are inferred by the following rules:
// - All columns are inferred to start with
// - All primary keys are subtracted from this set
// Update does not automatically update the record in case of default values. Use .Reload()
// to refresh the records.
func (o *User) Update(exec boil.Executor, whitelist ...string) error {
	var err error
	key := makeCacheKey(whitelist, nil)
	userUpdateCacheMut.RLock()
	cache, cached := userUpdateCache[key]
	userUpdateCacheMut.RUnlock()

	if !cached {
		wl := strmangle.UpdateColumnSet(userColumns, userPrimaryKeyColumns, whitelist)
		if len(wl) == 0 {
			return errors.New("models: unable to update users, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"users\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, userPrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(userType, userMapping, append(wl, userPrimaryKeyColumns...))
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
		return errors.Wrap(err, "models: unable to update users row")
	}

	if !cached {
		userUpdateCacheMut.Lock()
		userUpdateCache[key] = cache
		userUpdateCacheMut.Unlock()
	}

	return nil
}

// UpdateAllP updates all rows with matching column names, and panics on error.
func (q userQuery) UpdateAllP(cols M) {
	if err := q.UpdateAll(cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAll updates all rows with the specified column values.
func (q userQuery) UpdateAll(cols M) error {
	queries.SetUpdate(q.Query, cols)

	_, err := q.Query.Exec()
	if err != nil {
		return errors.Wrap(err, "models: unable to update all for users")
	}

	return nil
}

// UpdateAllG updates all rows with the specified column values.
func (o UserSlice) UpdateAllG(cols M) error {
	return o.UpdateAll(boil.GetDB(), cols)
}

// UpdateAllGP updates all rows with the specified column values, and panics on error.
func (o UserSlice) UpdateAllGP(cols M) {
	if err := o.UpdateAll(boil.GetDB(), cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAllP updates all rows with the specified column values, and panics on error.
func (o UserSlice) UpdateAllP(exec boil.Executor, cols M) {
	if err := o.UpdateAll(exec, cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o UserSlice) UpdateAll(exec boil.Executor, cols M) error {
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
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), userPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"UPDATE \"users\" SET %s WHERE (\"id\") IN (%s)",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(o)*len(userPrimaryKeyColumns), len(colNames)+1, len(userPrimaryKeyColumns)),
	)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all in user slice")
	}

	return nil
}

// UpsertG attempts an insert, and does an update or ignore on conflict.
func (o *User) UpsertG(updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) error {
	return o.Upsert(boil.GetDB(), updateOnConflict, conflictColumns, updateColumns, whitelist...)
}

// UpsertGP attempts an insert, and does an update or ignore on conflict. Panics on error.
func (o *User) UpsertGP(updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) {
	if err := o.Upsert(boil.GetDB(), updateOnConflict, conflictColumns, updateColumns, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpsertP attempts an insert using an executor, and does an update or ignore on conflict.
// UpsertP panics on error.
func (o *User) UpsertP(exec boil.Executor, updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) {
	if err := o.Upsert(exec, updateOnConflict, conflictColumns, updateColumns, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
func (o *User) Upsert(exec boil.Executor, updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) error {
	if o == nil {
		return errors.New("models: no users provided for upsert")
	}

	nzDefaults := queries.NonZeroDefaultSet(userColumnsWithDefault, o)

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

	userUpsertCacheMut.RLock()
	cache, cached := userUpsertCache[key]
	userUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		var ret []string
		whitelist, ret = strmangle.InsertColumnSet(
			userColumns,
			userColumnsWithDefault,
			userColumnsWithoutDefault,
			nzDefaults,
			whitelist,
		)
		update := strmangle.UpdateColumnSet(
			userColumns,
			userPrimaryKeyColumns,
			updateColumns,
		)
		if len(update) == 0 {
			return errors.New("models: unable to upsert users, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(userPrimaryKeyColumns))
			copy(conflict, userPrimaryKeyColumns)
		}
		cache.query = queries.BuildUpsertQueryPostgres(dialect, "\"users\"", updateOnConflict, ret, update, conflict, whitelist)

		cache.valueMapping, err = queries.BindMapping(userType, userMapping, whitelist)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(userType, userMapping, ret)
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
		return errors.Wrap(err, "models: unable to upsert users")
	}

	if !cached {
		userUpsertCacheMut.Lock()
		userUpsertCache[key] = cache
		userUpsertCacheMut.Unlock()
	}

	return nil
}

// DeleteP deletes a single User record with an executor.
// DeleteP will match against the primary key column to find the record to delete.
// Panics on error.
func (o *User) DeleteP(exec boil.Executor) {
	if err := o.Delete(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteG deletes a single User record.
// DeleteG will match against the primary key column to find the record to delete.
func (o *User) DeleteG() error {
	if o == nil {
		return errors.New("models: no User provided for deletion")
	}

	return o.Delete(boil.GetDB())
}

// DeleteGP deletes a single User record.
// DeleteGP will match against the primary key column to find the record to delete.
// Panics on error.
func (o *User) DeleteGP() {
	if err := o.DeleteG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Delete deletes a single User record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *User) Delete(exec boil.Executor) error {
	if o == nil {
		return errors.New("models: no User provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), userPrimaryKeyMapping)
	sql := "DELETE FROM \"users\" WHERE \"id\"=$1"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete from users")
	}

	return nil
}

// DeleteAllP deletes all rows, and panics on error.
func (q userQuery) DeleteAllP() {
	if err := q.DeleteAll(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAll deletes all matching rows.
func (q userQuery) DeleteAll() error {
	if q.Query == nil {
		return errors.New("models: no userQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	_, err := q.Query.Exec()
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from users")
	}

	return nil
}

// DeleteAllGP deletes all rows in the slice, and panics on error.
func (o UserSlice) DeleteAllGP() {
	if err := o.DeleteAllG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAllG deletes all rows in the slice.
func (o UserSlice) DeleteAllG() error {
	if o == nil {
		return errors.New("models: no User slice provided for delete all")
	}
	return o.DeleteAll(boil.GetDB())
}

// DeleteAllP deletes all rows in the slice, using an executor, and panics on error.
func (o UserSlice) DeleteAllP(exec boil.Executor) {
	if err := o.DeleteAll(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o UserSlice) DeleteAll(exec boil.Executor) error {
	if o == nil {
		return errors.New("models: no User slice provided for delete all")
	}

	if len(o) == 0 {
		return nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), userPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"DELETE FROM \"users\" WHERE (%s) IN (%s)",
		strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, userPrimaryKeyColumns), ","),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(o)*len(userPrimaryKeyColumns), 1, len(userPrimaryKeyColumns)),
	)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from user slice")
	}

	return nil
}

// ReloadGP refetches the object from the database and panics on error.
func (o *User) ReloadGP() {
	if err := o.ReloadG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadP refetches the object from the database with an executor. Panics on error.
func (o *User) ReloadP(exec boil.Executor) {
	if err := o.Reload(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadG refetches the object from the database using the primary keys.
func (o *User) ReloadG() error {
	if o == nil {
		return errors.New("models: no User provided for reload")
	}

	return o.Reload(boil.GetDB())
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *User) Reload(exec boil.Executor) error {
	ret, err := FindUser(exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAllGP refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
// Panics on error.
func (o *UserSlice) ReloadAllGP() {
	if err := o.ReloadAllG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadAllP refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
// Panics on error.
func (o *UserSlice) ReloadAllP(exec boil.Executor) {
	if err := o.ReloadAll(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadAllG refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *UserSlice) ReloadAllG() error {
	if o == nil {
		return errors.New("models: empty UserSlice provided for reload all")
	}

	return o.ReloadAll(boil.GetDB())
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *UserSlice) ReloadAll(exec boil.Executor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	users := UserSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), userPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"SELECT \"users\".* FROM \"users\" WHERE (%s) IN (%s)",
		strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, userPrimaryKeyColumns), ","),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(*o)*len(userPrimaryKeyColumns), 1, len(userPrimaryKeyColumns)),
	)

	q := queries.Raw(exec, sql, args...)

	err := q.Bind(&users)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in UserSlice")
	}

	*o = users

	return nil
}

// UserExists checks if the User row exists.
func UserExists(exec boil.Executor, id int64) (bool, error) {
	var exists bool

	sql := "select exists(select 1 from \"users\" where \"id\"=$1 limit 1)"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, id)
	}

	row := exec.QueryRow(sql, id)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if users exists")
	}

	return exists, nil
}

// UserExistsG checks if the User row exists.
func UserExistsG(id int64) (bool, error) {
	return UserExists(boil.GetDB(), id)
}

// UserExistsGP checks if the User row exists. Panics on error.
func UserExistsGP(id int64) bool {
	e, err := UserExists(boil.GetDB(), id)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// UserExistsP checks if the User row exists. Panics on error.
func UserExistsP(exec boil.Executor, id int64) bool {
	e, err := UserExists(exec, id)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}
