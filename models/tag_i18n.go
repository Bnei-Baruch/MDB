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

// TagI18n is an object representing the database table.
type TagI18n struct {
	TagID            int64       `boil:"tag_id" json:"tag_id" toml:"tag_id" yaml:"tag_id"`
	Language         string      `boil:"language" json:"language" toml:"language" yaml:"language"`
	OriginalLanguage null.String `boil:"original_language" json:"original_language,omitempty" toml:"original_language" yaml:"original_language,omitempty"`
	Label            null.String `boil:"label" json:"label,omitempty" toml:"label" yaml:"label,omitempty"`
	UserID           null.Int64  `boil:"user_id" json:"user_id,omitempty" toml:"user_id" yaml:"user_id,omitempty"`
	CreatedAt        time.Time   `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`

	R *tagI18nR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L tagI18nL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

// tagI18nR is where relationships are stored.
type tagI18nR struct {
	Tag  *Tag
	User *User
}

// tagI18nL is where Load methods for each relationship are stored.
type tagI18nL struct{}

var (
	tagI18nColumns               = []string{"tag_id", "language", "original_language", "label", "user_id", "created_at"}
	tagI18nColumnsWithoutDefault = []string{"tag_id", "language", "original_language", "label", "user_id"}
	tagI18nColumnsWithDefault    = []string{"created_at"}
	tagI18nPrimaryKeyColumns     = []string{"tag_id", "language"}
)

type (
	// TagI18nSlice is an alias for a slice of pointers to TagI18n.
	// This should generally be used opposed to []TagI18n.
	TagI18nSlice []*TagI18n

	tagI18nQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	tagI18nType                 = reflect.TypeOf(&TagI18n{})
	tagI18nMapping              = queries.MakeStructMapping(tagI18nType)
	tagI18nPrimaryKeyMapping, _ = queries.BindMapping(tagI18nType, tagI18nMapping, tagI18nPrimaryKeyColumns)
	tagI18nInsertCacheMut       sync.RWMutex
	tagI18nInsertCache          = make(map[string]insertCache)
	tagI18nUpdateCacheMut       sync.RWMutex
	tagI18nUpdateCache          = make(map[string]updateCache)
	tagI18nUpsertCacheMut       sync.RWMutex
	tagI18nUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force bytes in case of primary key column that uses []byte (for relationship compares)
	_ = bytes.MinRead
)

// OneP returns a single tagI18n record from the query, and panics on error.
func (q tagI18nQuery) OneP() *TagI18n {
	o, err := q.One()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// One returns a single tagI18n record from the query.
func (q tagI18nQuery) One() (*TagI18n, error) {
	o := &TagI18n{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(o)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for tag_i18n")
	}

	return o, nil
}

// AllP returns all TagI18n records from the query, and panics on error.
func (q tagI18nQuery) AllP() TagI18nSlice {
	o, err := q.All()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// All returns all TagI18n records from the query.
func (q tagI18nQuery) All() (TagI18nSlice, error) {
	var o TagI18nSlice

	err := q.Bind(&o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to TagI18n slice")
	}

	return o, nil
}

// CountP returns the count of all TagI18n records in the query, and panics on error.
func (q tagI18nQuery) CountP() int64 {
	c, err := q.Count()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return c
}

// Count returns the count of all TagI18n records in the query.
func (q tagI18nQuery) Count() (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRow().Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count tag_i18n rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table, and panics on error.
func (q tagI18nQuery) ExistsP() bool {
	e, err := q.Exists()
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// Exists checks if the row exists in the table.
func (q tagI18nQuery) Exists() (bool, error) {
	var count int64

	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRow().Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if tag_i18n exists")
	}

	return count > 0, nil
}

// TagG pointed to by the foreign key.
func (o *TagI18n) TagG(mods ...qm.QueryMod) tagQuery {
	return o.Tag(boil.GetDB(), mods...)
}

// Tag pointed to by the foreign key.
func (o *TagI18n) Tag(exec boil.Executor, mods ...qm.QueryMod) tagQuery {
	queryMods := []qm.QueryMod{
		qm.Where("id=?", o.TagID),
	}

	queryMods = append(queryMods, mods...)

	query := Tags(exec, queryMods...)
	queries.SetFrom(query.Query, "\"tags\"")

	return query
}

// UserG pointed to by the foreign key.
func (o *TagI18n) UserG(mods ...qm.QueryMod) userQuery {
	return o.User(boil.GetDB(), mods...)
}

// User pointed to by the foreign key.
func (o *TagI18n) User(exec boil.Executor, mods ...qm.QueryMod) userQuery {
	queryMods := []qm.QueryMod{
		qm.Where("id=?", o.UserID),
	}

	queryMods = append(queryMods, mods...)

	query := Users(exec, queryMods...)
	queries.SetFrom(query.Query, "\"users\"")

	return query
}

// LoadTag allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (tagI18nL) LoadTag(e boil.Executor, singular bool, maybeTagI18n interface{}) error {
	var slice []*TagI18n
	var object *TagI18n

	count := 1
	if singular {
		object = maybeTagI18n.(*TagI18n)
	} else {
		slice = *maybeTagI18n.(*TagI18nSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &tagI18nR{}
		}
		args[0] = object.TagID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &tagI18nR{}
			}
			args[i] = obj.TagID
		}
	}

	query := fmt.Sprintf(
		"select * from \"tags\" where \"id\" in (%s)",
		strmangle.Placeholders(dialect.IndexPlaceholders, count, 1, 1),
	)

	if boil.DebugMode {
		fmt.Fprintf(boil.DebugWriter, "%s\n%v\n", query, args)
	}

	results, err := e.Query(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to eager load Tag")
	}
	defer results.Close()

	var resultSlice []*Tag
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice Tag")
	}

	if singular && len(resultSlice) != 0 {
		object.R.Tag = resultSlice[0]
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.TagID == foreign.ID {
				local.R.Tag = foreign
				break
			}
		}
	}

	return nil
}

// LoadUser allows an eager lookup of values, cached into the
// loaded structs of the objects.
func (tagI18nL) LoadUser(e boil.Executor, singular bool, maybeTagI18n interface{}) error {
	var slice []*TagI18n
	var object *TagI18n

	count := 1
	if singular {
		object = maybeTagI18n.(*TagI18n)
	} else {
		slice = *maybeTagI18n.(*TagI18nSlice)
		count = len(slice)
	}

	args := make([]interface{}, count)
	if singular {
		if object.R == nil {
			object.R = &tagI18nR{}
		}
		args[0] = object.UserID
	} else {
		for i, obj := range slice {
			if obj.R == nil {
				obj.R = &tagI18nR{}
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

// SetTagG of the tag_i18n to the related item.
// Sets o.R.Tag to related.
// Adds o to related.R.TagI18ns.
// Uses the global database handle.
func (o *TagI18n) SetTagG(insert bool, related *Tag) error {
	return o.SetTag(boil.GetDB(), insert, related)
}

// SetTagP of the tag_i18n to the related item.
// Sets o.R.Tag to related.
// Adds o to related.R.TagI18ns.
// Panics on error.
func (o *TagI18n) SetTagP(exec boil.Executor, insert bool, related *Tag) {
	if err := o.SetTag(exec, insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetTagGP of the tag_i18n to the related item.
// Sets o.R.Tag to related.
// Adds o to related.R.TagI18ns.
// Uses the global database handle and panics on error.
func (o *TagI18n) SetTagGP(insert bool, related *Tag) {
	if err := o.SetTag(boil.GetDB(), insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetTag of the tag_i18n to the related item.
// Sets o.R.Tag to related.
// Adds o to related.R.TagI18ns.
func (o *TagI18n) SetTag(exec boil.Executor, insert bool, related *Tag) error {
	var err error
	if insert {
		if err = related.Insert(exec); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE \"tag_i18n\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, []string{"tag_id"}),
		strmangle.WhereClause("\"", "\"", 2, tagI18nPrimaryKeyColumns),
	)
	values := []interface{}{related.ID, o.TagID, o.Language}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, updateQuery)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	if _, err = exec.Exec(updateQuery, values...); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	o.TagID = related.ID

	if o.R == nil {
		o.R = &tagI18nR{
			Tag: related,
		}
	} else {
		o.R.Tag = related
	}

	if related.R == nil {
		related.R = &tagR{
			TagI18ns: TagI18nSlice{o},
		}
	} else {
		related.R.TagI18ns = append(related.R.TagI18ns, o)
	}

	return nil
}

// SetUserG of the tag_i18n to the related item.
// Sets o.R.User to related.
// Adds o to related.R.TagI18ns.
// Uses the global database handle.
func (o *TagI18n) SetUserG(insert bool, related *User) error {
	return o.SetUser(boil.GetDB(), insert, related)
}

// SetUserP of the tag_i18n to the related item.
// Sets o.R.User to related.
// Adds o to related.R.TagI18ns.
// Panics on error.
func (o *TagI18n) SetUserP(exec boil.Executor, insert bool, related *User) {
	if err := o.SetUser(exec, insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetUserGP of the tag_i18n to the related item.
// Sets o.R.User to related.
// Adds o to related.R.TagI18ns.
// Uses the global database handle and panics on error.
func (o *TagI18n) SetUserGP(insert bool, related *User) {
	if err := o.SetUser(boil.GetDB(), insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetUser of the tag_i18n to the related item.
// Sets o.R.User to related.
// Adds o to related.R.TagI18ns.
func (o *TagI18n) SetUser(exec boil.Executor, insert bool, related *User) error {
	var err error
	if insert {
		if err = related.Insert(exec); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE \"tag_i18n\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, []string{"user_id"}),
		strmangle.WhereClause("\"", "\"", 2, tagI18nPrimaryKeyColumns),
	)
	values := []interface{}{related.ID, o.TagID, o.Language}

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
		o.R = &tagI18nR{
			User: related,
		}
	} else {
		o.R.User = related
	}

	if related.R == nil {
		related.R = &userR{
			TagI18ns: TagI18nSlice{o},
		}
	} else {
		related.R.TagI18ns = append(related.R.TagI18ns, o)
	}

	return nil
}

// RemoveUserG relationship.
// Sets o.R.User to nil.
// Removes o from all passed in related items' relationships struct (Optional).
// Uses the global database handle.
func (o *TagI18n) RemoveUserG(related *User) error {
	return o.RemoveUser(boil.GetDB(), related)
}

// RemoveUserP relationship.
// Sets o.R.User to nil.
// Removes o from all passed in related items' relationships struct (Optional).
// Panics on error.
func (o *TagI18n) RemoveUserP(exec boil.Executor, related *User) {
	if err := o.RemoveUser(exec, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveUserGP relationship.
// Sets o.R.User to nil.
// Removes o from all passed in related items' relationships struct (Optional).
// Uses the global database handle and panics on error.
func (o *TagI18n) RemoveUserGP(related *User) {
	if err := o.RemoveUser(boil.GetDB(), related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveUser relationship.
// Sets o.R.User to nil.
// Removes o from all passed in related items' relationships struct (Optional).
func (o *TagI18n) RemoveUser(exec boil.Executor, related *User) error {
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

	for i, ri := range related.R.TagI18ns {
		if o.UserID.Int64 != ri.UserID.Int64 {
			continue
		}

		ln := len(related.R.TagI18ns)
		if ln > 1 && i < ln-1 {
			related.R.TagI18ns[i] = related.R.TagI18ns[ln-1]
		}
		related.R.TagI18ns = related.R.TagI18ns[:ln-1]
		break
	}
	return nil
}

// TagI18nsG retrieves all records.
func TagI18nsG(mods ...qm.QueryMod) tagI18nQuery {
	return TagI18ns(boil.GetDB(), mods...)
}

// TagI18ns retrieves all the records using an executor.
func TagI18ns(exec boil.Executor, mods ...qm.QueryMod) tagI18nQuery {
	mods = append(mods, qm.From("\"tag_i18n\""))
	return tagI18nQuery{NewQuery(exec, mods...)}
}

// FindTagI18nG retrieves a single record by ID.
func FindTagI18nG(tagID int64, language string, selectCols ...string) (*TagI18n, error) {
	return FindTagI18n(boil.GetDB(), tagID, language, selectCols...)
}

// FindTagI18nGP retrieves a single record by ID, and panics on error.
func FindTagI18nGP(tagID int64, language string, selectCols ...string) *TagI18n {
	retobj, err := FindTagI18n(boil.GetDB(), tagID, language, selectCols...)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return retobj
}

// FindTagI18n retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindTagI18n(exec boil.Executor, tagID int64, language string, selectCols ...string) (*TagI18n, error) {
	tagI18nObj := &TagI18n{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"tag_i18n\" where \"tag_id\"=$1 AND \"language\"=$2", sel,
	)

	q := queries.Raw(exec, query, tagID, language)

	err := q.Bind(tagI18nObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from tag_i18n")
	}

	return tagI18nObj, nil
}

// FindTagI18nP retrieves a single record by ID with an executor, and panics on error.
func FindTagI18nP(exec boil.Executor, tagID int64, language string, selectCols ...string) *TagI18n {
	retobj, err := FindTagI18n(exec, tagID, language, selectCols...)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return retobj
}

// InsertG a single record. See Insert for whitelist behavior description.
func (o *TagI18n) InsertG(whitelist ...string) error {
	return o.Insert(boil.GetDB(), whitelist...)
}

// InsertGP a single record, and panics on error. See Insert for whitelist
// behavior description.
func (o *TagI18n) InsertGP(whitelist ...string) {
	if err := o.Insert(boil.GetDB(), whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// InsertP a single record using an executor, and panics on error. See Insert
// for whitelist behavior description.
func (o *TagI18n) InsertP(exec boil.Executor, whitelist ...string) {
	if err := o.Insert(exec, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Insert a single record using an executor.
// Whitelist behavior: If a whitelist is provided, only those columns supplied are inserted
// No whitelist behavior: Without a whitelist, columns are inferred by the following rules:
// - All columns without a default value are included (i.e. name, age)
// - All columns with a default, but non-zero are included (i.e. health = 75)
func (o *TagI18n) Insert(exec boil.Executor, whitelist ...string) error {
	if o == nil {
		return errors.New("models: no tag_i18n provided for insertion")
	}

	var err error

	nzDefaults := queries.NonZeroDefaultSet(tagI18nColumnsWithDefault, o)

	key := makeCacheKey(whitelist, nzDefaults)
	tagI18nInsertCacheMut.RLock()
	cache, cached := tagI18nInsertCache[key]
	tagI18nInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := strmangle.InsertColumnSet(
			tagI18nColumns,
			tagI18nColumnsWithDefault,
			tagI18nColumnsWithoutDefault,
			nzDefaults,
			whitelist,
		)

		cache.valueMapping, err = queries.BindMapping(tagI18nType, tagI18nMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(tagI18nType, tagI18nMapping, returnColumns)
		if err != nil {
			return err
		}
		cache.query = fmt.Sprintf("INSERT INTO \"tag_i18n\" (\"%s\") VALUES (%s)", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.IndexPlaceholders, len(wl), 1, 1))

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
		return errors.Wrap(err, "models: unable to insert into tag_i18n")
	}

	if !cached {
		tagI18nInsertCacheMut.Lock()
		tagI18nInsertCache[key] = cache
		tagI18nInsertCacheMut.Unlock()
	}

	return nil
}

// UpdateG a single TagI18n record. See Update for
// whitelist behavior description.
func (o *TagI18n) UpdateG(whitelist ...string) error {
	return o.Update(boil.GetDB(), whitelist...)
}

// UpdateGP a single TagI18n record.
// UpdateGP takes a whitelist of column names that should be updated.
// Panics on error. See Update for whitelist behavior description.
func (o *TagI18n) UpdateGP(whitelist ...string) {
	if err := o.Update(boil.GetDB(), whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateP uses an executor to update the TagI18n, and panics on error.
// See Update for whitelist behavior description.
func (o *TagI18n) UpdateP(exec boil.Executor, whitelist ...string) {
	err := o.Update(exec, whitelist...)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// Update uses an executor to update the TagI18n.
// Whitelist behavior: If a whitelist is provided, only the columns given are updated.
// No whitelist behavior: Without a whitelist, columns are inferred by the following rules:
// - All columns are inferred to start with
// - All primary keys are subtracted from this set
// Update does not automatically update the record in case of default values. Use .Reload()
// to refresh the records.
func (o *TagI18n) Update(exec boil.Executor, whitelist ...string) error {
	var err error
	key := makeCacheKey(whitelist, nil)
	tagI18nUpdateCacheMut.RLock()
	cache, cached := tagI18nUpdateCache[key]
	tagI18nUpdateCacheMut.RUnlock()

	if !cached {
		wl := strmangle.UpdateColumnSet(tagI18nColumns, tagI18nPrimaryKeyColumns, whitelist)
		if len(wl) == 0 {
			return errors.New("models: unable to update tag_i18n, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"tag_i18n\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, tagI18nPrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(tagI18nType, tagI18nMapping, append(wl, tagI18nPrimaryKeyColumns...))
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
		return errors.Wrap(err, "models: unable to update tag_i18n row")
	}

	if !cached {
		tagI18nUpdateCacheMut.Lock()
		tagI18nUpdateCache[key] = cache
		tagI18nUpdateCacheMut.Unlock()
	}

	return nil
}

// UpdateAllP updates all rows with matching column names, and panics on error.
func (q tagI18nQuery) UpdateAllP(cols M) {
	if err := q.UpdateAll(cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAll updates all rows with the specified column values.
func (q tagI18nQuery) UpdateAll(cols M) error {
	queries.SetUpdate(q.Query, cols)

	_, err := q.Query.Exec()
	if err != nil {
		return errors.Wrap(err, "models: unable to update all for tag_i18n")
	}

	return nil
}

// UpdateAllG updates all rows with the specified column values.
func (o TagI18nSlice) UpdateAllG(cols M) error {
	return o.UpdateAll(boil.GetDB(), cols)
}

// UpdateAllGP updates all rows with the specified column values, and panics on error.
func (o TagI18nSlice) UpdateAllGP(cols M) {
	if err := o.UpdateAll(boil.GetDB(), cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAllP updates all rows with the specified column values, and panics on error.
func (o TagI18nSlice) UpdateAllP(exec boil.Executor, cols M) {
	if err := o.UpdateAll(exec, cols); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o TagI18nSlice) UpdateAll(exec boil.Executor, cols M) error {
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
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), tagI18nPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"UPDATE \"tag_i18n\" SET %s WHERE (\"tag_id\",\"language\") IN (%s)",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(o)*len(tagI18nPrimaryKeyColumns), len(colNames)+1, len(tagI18nPrimaryKeyColumns)),
	)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all in tagI18n slice")
	}

	return nil
}

// UpsertG attempts an insert, and does an update or ignore on conflict.
func (o *TagI18n) UpsertG(updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) error {
	return o.Upsert(boil.GetDB(), updateOnConflict, conflictColumns, updateColumns, whitelist...)
}

// UpsertGP attempts an insert, and does an update or ignore on conflict. Panics on error.
func (o *TagI18n) UpsertGP(updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) {
	if err := o.Upsert(boil.GetDB(), updateOnConflict, conflictColumns, updateColumns, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpsertP attempts an insert using an executor, and does an update or ignore on conflict.
// UpsertP panics on error.
func (o *TagI18n) UpsertP(exec boil.Executor, updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) {
	if err := o.Upsert(exec, updateOnConflict, conflictColumns, updateColumns, whitelist...); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
func (o *TagI18n) Upsert(exec boil.Executor, updateOnConflict bool, conflictColumns []string, updateColumns []string, whitelist ...string) error {
	if o == nil {
		return errors.New("models: no tag_i18n provided for upsert")
	}

	nzDefaults := queries.NonZeroDefaultSet(tagI18nColumnsWithDefault, o)

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

	tagI18nUpsertCacheMut.RLock()
	cache, cached := tagI18nUpsertCache[key]
	tagI18nUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		var ret []string
		whitelist, ret = strmangle.InsertColumnSet(
			tagI18nColumns,
			tagI18nColumnsWithDefault,
			tagI18nColumnsWithoutDefault,
			nzDefaults,
			whitelist,
		)
		update := strmangle.UpdateColumnSet(
			tagI18nColumns,
			tagI18nPrimaryKeyColumns,
			updateColumns,
		)
		if len(update) == 0 {
			return errors.New("models: unable to upsert tag_i18n, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(tagI18nPrimaryKeyColumns))
			copy(conflict, tagI18nPrimaryKeyColumns)
		}
		cache.query = queries.BuildUpsertQueryPostgres(dialect, "\"tag_i18n\"", updateOnConflict, ret, update, conflict, whitelist)

		cache.valueMapping, err = queries.BindMapping(tagI18nType, tagI18nMapping, whitelist)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(tagI18nType, tagI18nMapping, ret)
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
		return errors.Wrap(err, "models: unable to upsert tag_i18n")
	}

	if !cached {
		tagI18nUpsertCacheMut.Lock()
		tagI18nUpsertCache[key] = cache
		tagI18nUpsertCacheMut.Unlock()
	}

	return nil
}

// DeleteP deletes a single TagI18n record with an executor.
// DeleteP will match against the primary key column to find the record to delete.
// Panics on error.
func (o *TagI18n) DeleteP(exec boil.Executor) {
	if err := o.Delete(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteG deletes a single TagI18n record.
// DeleteG will match against the primary key column to find the record to delete.
func (o *TagI18n) DeleteG() error {
	if o == nil {
		return errors.New("models: no TagI18n provided for deletion")
	}

	return o.Delete(boil.GetDB())
}

// DeleteGP deletes a single TagI18n record.
// DeleteGP will match against the primary key column to find the record to delete.
// Panics on error.
func (o *TagI18n) DeleteGP() {
	if err := o.DeleteG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Delete deletes a single TagI18n record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *TagI18n) Delete(exec boil.Executor) error {
	if o == nil {
		return errors.New("models: no TagI18n provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), tagI18nPrimaryKeyMapping)
	sql := "DELETE FROM \"tag_i18n\" WHERE \"tag_id\"=$1 AND \"language\"=$2"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete from tag_i18n")
	}

	return nil
}

// DeleteAllP deletes all rows, and panics on error.
func (q tagI18nQuery) DeleteAllP() {
	if err := q.DeleteAll(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAll deletes all matching rows.
func (q tagI18nQuery) DeleteAll() error {
	if q.Query == nil {
		return errors.New("models: no tagI18nQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	_, err := q.Query.Exec()
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from tag_i18n")
	}

	return nil
}

// DeleteAllGP deletes all rows in the slice, and panics on error.
func (o TagI18nSlice) DeleteAllGP() {
	if err := o.DeleteAllG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAllG deletes all rows in the slice.
func (o TagI18nSlice) DeleteAllG() error {
	if o == nil {
		return errors.New("models: no TagI18n slice provided for delete all")
	}
	return o.DeleteAll(boil.GetDB())
}

// DeleteAllP deletes all rows in the slice, using an executor, and panics on error.
func (o TagI18nSlice) DeleteAllP(exec boil.Executor) {
	if err := o.DeleteAll(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o TagI18nSlice) DeleteAll(exec boil.Executor) error {
	if o == nil {
		return errors.New("models: no TagI18n slice provided for delete all")
	}

	if len(o) == 0 {
		return nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), tagI18nPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"DELETE FROM \"tag_i18n\" WHERE (%s) IN (%s)",
		strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, tagI18nPrimaryKeyColumns), ","),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(o)*len(tagI18nPrimaryKeyColumns), 1, len(tagI18nPrimaryKeyColumns)),
	)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args)
	}

	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from tagI18n slice")
	}

	return nil
}

// ReloadGP refetches the object from the database and panics on error.
func (o *TagI18n) ReloadGP() {
	if err := o.ReloadG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadP refetches the object from the database with an executor. Panics on error.
func (o *TagI18n) ReloadP(exec boil.Executor) {
	if err := o.Reload(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadG refetches the object from the database using the primary keys.
func (o *TagI18n) ReloadG() error {
	if o == nil {
		return errors.New("models: no TagI18n provided for reload")
	}

	return o.Reload(boil.GetDB())
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *TagI18n) Reload(exec boil.Executor) error {
	ret, err := FindTagI18n(exec, o.TagID, o.Language)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAllGP refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
// Panics on error.
func (o *TagI18nSlice) ReloadAllGP() {
	if err := o.ReloadAllG(); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadAllP refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
// Panics on error.
func (o *TagI18nSlice) ReloadAllP(exec boil.Executor) {
	if err := o.ReloadAll(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadAllG refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *TagI18nSlice) ReloadAllG() error {
	if o == nil {
		return errors.New("models: empty TagI18nSlice provided for reload all")
	}

	return o.ReloadAll(boil.GetDB())
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *TagI18nSlice) ReloadAll(exec boil.Executor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	tagI18ns := TagI18nSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), tagI18nPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf(
		"SELECT \"tag_i18n\".* FROM \"tag_i18n\" WHERE (%s) IN (%s)",
		strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, tagI18nPrimaryKeyColumns), ","),
		strmangle.Placeholders(dialect.IndexPlaceholders, len(*o)*len(tagI18nPrimaryKeyColumns), 1, len(tagI18nPrimaryKeyColumns)),
	)

	q := queries.Raw(exec, sql, args...)

	err := q.Bind(&tagI18ns)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in TagI18nSlice")
	}

	*o = tagI18ns

	return nil
}

// TagI18nExists checks if the TagI18n row exists.
func TagI18nExists(exec boil.Executor, tagID int64, language string) (bool, error) {
	var exists bool

	sql := "select exists(select 1 from \"tag_i18n\" where \"tag_id\"=$1 AND \"language\"=$2 limit 1)"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, tagID, language)
	}

	row := exec.QueryRow(sql, tagID, language)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if tag_i18n exists")
	}

	return exists, nil
}

// TagI18nExistsG checks if the TagI18n row exists.
func TagI18nExistsG(tagID int64, language string) (bool, error) {
	return TagI18nExists(boil.GetDB(), tagID, language)
}

// TagI18nExistsGP checks if the TagI18n row exists. Panics on error.
func TagI18nExistsGP(tagID int64, language string) bool {
	e, err := TagI18nExists(boil.GetDB(), tagID, language)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// TagI18nExistsP checks if the TagI18n row exists. Panics on error.
func TagI18nExistsP(exec boil.Executor, tagID int64, language string) bool {
	e, err := TagI18nExists(exec, tagID, language)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}