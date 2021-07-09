package api

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/volatiletech/sqlboiler/types"
	"gopkg.in/volatiletech/null.v6"

	"github.com/Bnei-Baruch/mdb/models"
)

type (
	// Common

	Operation struct {
		Station    string `json:"station" binding:"required"`
		User       string `json:"user" binding:"required,email"`
		WorkflowID string `json:"workflow_id"`
	}

	File struct {
		FileName  string     `json:"file_name" binding:"required,max=255"`
		Sha1      string     `json:"sha1" binding:"required,len=40,hexadecimal"`
		Size      int64      `json:"size" binding:"required"`
		CreatedAt *Timestamp `json:"created_at" binding:"required"`
		Type      string     `json:"type" binding:"max=16"`
		SubType   string     `json:"sub_type" binding:"max=16"`
		MimeType  string     `json:"mime_type" binding:"max=255"`
		Language  string     `json:"language" binding:"omitempty,min=2,max=3"`
	}

	// MaybeFile same as File but all fields are optional
	MaybeFile struct {
		FileName  string     `json:"file_name" binding:"omitempty,max=255"`
		Sha1      string     `json:"sha1" binding:"omitempty,len=40,hexadecimal"`
		Size      int64      `json:"size" binding:"omitempty"`
		CreatedAt *Timestamp `json:"created_at" binding:"omitempty"`
		Type      string     `json:"type" binding:"max=16"`
		SubType   string     `json:"sub_type" binding:"max=16"`
		MimeType  string     `json:"mime_type" binding:"max=255"`
		Language  string     `json:"language" binding:"omitempty,min=2,max=3"`
	}

	AVFile struct {
		File
		Duration  float64 `json:"duration"`
		VideoSize string  `json:"video_size"`
	}

	CITMetadataMajor struct {
		Type string `json:"type" binding:"omitempty,eq=source|eq=tag|eq=likutim"`
		Idx  int    `json:"idx" binding:"omitempty,gte=0"`
	}
	CITMetadata struct {
		ContentType    string            `json:"content_type" binding:"required"`
		CaptureDate    Date              `json:"capture_date" binding:"required"`
		FinalName      string            `json:"final_name" binding:"required,max=255"`
		Language       string            `json:"language" binding:"required,min=2,max=3"`
		Lecturer       string            `json:"lecturer" binding:"required"`
		AutoName       string            `json:"auto_name"`
		ManualName     string            `json:"manual_name"`
		WeekDate       *Date             `json:"week_date"`
		Number         null.Int          `json:"number"`
		Part           null.Int          `json:"part"`
		Sources        []string          `json:"sources" binding:"omitempty,dive,len=8"`
		Tags           []string          `json:"tags" binding:"omitempty,dive,len=8"`
		Likutim        []string          `json:"likutims" binding:"omitempty,dive,len=8"`
		ArtifactType   null.String       `json:"artifact_type"`
		HasTranslation bool              `json:"has_translation"`
		RequireTest    bool              `json:"require_test"`
		CollectionUID  null.String       `json:"collection_uid" binding:"omitempty,len=8"`
		Episode        null.String       `json:"episode"`
		PartType       null.Int          `json:"part_type"`
		Major          *CITMetadataMajor `json:"major" binding:"omitempty"`
		LabelID        null.Int          `json:"label_id"`
		FilmDate       *Date             `json:"film_date"`
		UnitToFixUID   null.String       `json:"fix_unit_uid" binding:"omitempty,len=8"`
	}

	Rename struct {
		Sha1     string `json:"sha1" binding:"required,len=40,hexadecimal"`
		FileName string `json:"file_name" binding:"required,max=255"`
	}

	// Operations

	CaptureStartRequest struct {
		Operation
		FileName      string `json:"file_name" binding:"max=255"`
		CaptureSource string `json:"capture_source"`
		CollectionUID string `json:"collection_uid"`
	}

	CaptureStopRequest struct {
		Operation
		File
		CaptureSource string   `json:"capture_source"`
		CollectionUID string   `json:"collection_uid"`
		Part          string   `json:"part"`
		LabelID       null.Int `json:"label_id"`
	}

	DemuxRequest struct {
		Operation
		Sha1          string  `json:"sha1" binding:"required,len=40,hexadecimal"`
		Original      AVFile  `json:"original"`
		Proxy         *AVFile `json:"proxy"`
		CaptureSource string  `json:"capture_source"`
	}

	TrimRequest struct {
		Operation
		OriginalSha1  string    `json:"original_sha1" binding:"required,len=40,hexadecimal"`
		ProxySha1     string    `json:"proxy_sha1" binding:"omitempty,len=40,hexadecimal"`
		Original      AVFile    `json:"original"`
		Proxy         *AVFile   `json:"proxy"`
		In            []float64 `json:"in"`
		Out           []float64 `json:"out"`
		CaptureSource string    `json:"capture_source"`
	}

	SendRequest struct {
		Operation
		Original Rename      `json:"original"`
		Proxy    *Rename     `json:"proxy"`
		Metadata CITMetadata `json:"metadata"`
		Mode     null.String `json:"mode"`
	}

	ConvertRequest struct {
		Operation
		Sha1   string   `json:"sha1" binding:"required,len=40,hexadecimal"`
		Output []AVFile `json:"output"`
	}

	UploadRequest struct {
		Operation
		AVFile
		Url string `json:"url" binding:"required"`
	}

	SirtutimRequest struct {
		Operation
		File
		OriginalSha1 string `json:"original_sha1" binding:"omitempty,len=40,hexadecimal"`
	}

	InsertRequest struct {
		Operation
		AVFile
		InsertType     string       `json:"insert_type" binding:"required"`
		Mode           string       `json:"mode" binding:"required"`
		ContentUnitUID string       `json:"content_unit_uid" binding:"omitempty,len=8"`
		ParentSha1     string       `json:"parent_sha1" binding:"omitempty,len=40,hexadecimal"`
		OldSha1        string       `json:"old_sha1" binding:"omitempty,len=40,hexadecimal"`
		PublisherUID   string       `json:"publisher_uid" binding:"omitempty,len=8"`
		Metadata       *CITMetadata `json:"metadata" binding:"omitempty"`
	}

	TranscodeRequest struct {
		Operation
		MaybeFile
		OriginalSha1 string `json:"original_sha1" binding:"omitempty,len=40,hexadecimal"`
		Message      string `json:"message" binding:"omitempty"`
	}

	JoinRequest struct {
		Operation
		OriginalShas []string `json:"original_shas" binding:"required,dive,len=40,hexadecimal"`
		ProxyShas    []string `json:"proxy_shas" binding:"omitempty,dive,len=40,hexadecimal"`
		Original     AVFile   `json:"original"`
		Proxy        *AVFile  `json:"proxy"`
	}

	// REST

	ListRequest struct {
		PageNumber int    `json:"page_no" form:"page_no" binding:"omitempty,min=1"`
		PageSize   int    `json:"page_size" form:"page_size" binding:"omitempty,min=1"`
		StartIndex int    `json:"start_index" form:"start_index" binding:"omitempty,min=1"`
		StopIndex  int    `json:"stop_index" form:"stop_index" binding:"omitempty,min=1"`
		OrderBy    string `json:"order_by" form:"order_by" binding:"omitempty"`
	}

	ListResponse struct {
		Total int64 `json:"total"`
	}

	IDsFilter struct {
		IDs []int64 `json:"ids" form:"id" binding:"omitempty"`
	}

	UIDsFilter struct {
		UIDs []string `json:"uids" form:"uid" binding:"omitempty"`
	}

	SHA1sFilter struct {
		SHA1s []string `json:"sha1s" form:"sha1" binding:"omitempty"`
	}

	PatternsFilter struct {
		Patterns []string `json:"patterns" form:"pattern" binding:"omitempty"`
	}

	ContentTypesFilter struct {
		ContentTypes []string `json:"content_types" form:"content_type" binding:"omitempty"`
	}

	OperationTypesFilter struct {
		OperationTypes []string `json:"operation_types" form:"operation_type" binding:"omitempty"`
	}

	SourcesFilter struct {
		Authors []string `json:"authors" form:"author" binding:"omitempty"`
		Sources []int64  `json:"sources" form:"source" binding:"omitempty"`
	}

	TagsFilter struct {
		Tags []int64 `json:"tags" form:"tag" binding:"omitempty"`
	}

	SearchTermFilter struct {
		Query string `json:"query" form:"query" binding:"omitempty"`
	}

	DateRangeFilter struct {
		StartDate string `json:"start_date" form:"start_date" binding:"omitempty"`
		EndDate   string `json:"end_date" form:"end_date" binding:"omitempty"`
	}

	SecureFilter struct {
		Levels []int16 `json:"security_levels" form:"secure" binding:"omitempty"`
	}

	PublishedFilter struct {
		Published string `json:"published" form:"published" binding:"omitempty"`
	}

	CollectionsRequest struct {
		ListRequest
		IDsFilter
		UIDsFilter
		ContentTypesFilter
		DateRangeFilter
		SecureFilter
		PublishedFilter
		SearchTermFilter
	}

	CollectionsResponse struct {
		ListResponse
		Collections []*Collection `json:"data"`
	}

	ContentUnitsRequest struct {
		ListRequest
		IDsFilter
		UIDsFilter
		ContentTypesFilter
		DateRangeFilter
		SecureFilter
		PublishedFilter
		SourcesFilter
		TagsFilter
		SearchTermFilter
	}

	ContentUnitsResponse struct {
		ListResponse
		ContentUnits []*ContentUnit `json:"data"`
	}

	FilesRequest struct {
		ListRequest
		IDsFilter
		UIDsFilter
		SHA1sFilter
		DateRangeFilter
		SecureFilter
		PublishedFilter
		SearchTermFilter
	}

	FilesResponse struct {
		ListResponse
		Files []*MFile `json:"data"`
	}

	OperationsRequest struct {
		ListRequest
		DateRangeFilter
		OperationTypesFilter
	}

	OperationsResponse struct {
		ListResponse
		Operations []*models.Operation `json:"data"`
	}

	AuthorsResponse struct {
		ListResponse
		Authors []*Author `json:"data"`
	}

	SourcesRequest struct {
		ListRequest
	}

	SourcesResponse struct {
		ListResponse
		Sources []*Source `json:"data"`
	}

	CreateSourceRequest struct {
		Source
		AuthorID null.Int64 `json:"author"`
	}

	TagsRequest struct {
		ListRequest
	}

	TagsResponse struct {
		ListResponse
		Tags []*Tag `json:"data"`
	}

	PersonsRequest struct {
		ListRequest
		IDsFilter
		UIDsFilter
		PatternsFilter
	}

	PersonsResponse struct {
		ListResponse
		Persons []*Person `json:"data"`
	}

	StoragesRequest struct {
		ListRequest
	}

	StoragesResponse struct {
		ListResponse
		Storages []*models.Storage `json:"data"`
	}

	PublishersRequest struct {
		ListRequest
		IDsFilter
		UIDsFilter
		PatternsFilter
	}

	PublishersResponse struct {
		ListResponse
		Publishers []*Publisher `json:"data"`
	}

	HierarchyRequest struct {
		Language string `json:"language" form:"language" binding:"omitempty,len=2"`
		RootUID  string `json:"root" form:"root" binding:"omitempty,len=8"`
		Depth    int    `json:"depth" form:"depth"`
	}

	SourcesHierarchyRequest struct {
		HierarchyRequest
	}

	TagsHierarchyRequest struct {
		HierarchyRequest
	}

	Collection struct {
		models.Collection
		I18n map[string]*models.CollectionI18n `json:"i18n"`
	}

	PartialCollection struct {
		models.Collection
		Secure null.Int16 `json:"secure"`
	}

	ContentUnit struct {
		models.ContentUnit
		I18n map[string]*models.ContentUnitI18n `json:"i18n"`
	}

	PartialContentUnit struct {
		models.ContentUnit
		Secure null.Int16 `json:"secure"`
	}

	CollectionContentUnit struct {
		Collection  *Collection  `json:"collection,omitempty"`
		ContentUnit *ContentUnit `json:"content_unit,omitempty"`
		Name        string       `json:"name"`
		Position    int          `json:"position"`
	}

	ContentUnitDerivation struct {
		Source  *ContentUnit `json:"source,omitempty"`
		Derived *ContentUnit `json:"derived,omitempty"`
		Name    string       `json:"name"`
	}

	// Marshalable File
	MFile struct {
		models.File
		Sha1Str      string           `json:"sha1"`
		OperationIds types.Int64Array `json:"operations"`
	}

	PartialFile struct {
		models.File
		Type    null.String `json:"type,omitempty"`
		SubType null.String `json:"sub_type,omitempty"`
		Secure  null.Int16  `json:"secure"`
	}

	Author struct {
		models.Author
		I18n    map[string]*models.AuthorI18n `json:"i18n"`
		Sources []*Source                     `json:"sources"`
	}

	Source struct {
		models.Source
		I18n map[string]*models.SourceI18n `json:"i18n"`
	}

	Tag struct {
		models.Tag
		I18n map[string]*models.TagI18n `json:"i18n"`
	}

	Person struct {
		models.Person
		I18n map[string]*models.PersonI18n `json:"i18n"`
	}

	ContentUnitPerson struct {
		ContentUnit *ContentUnit `json:"content_unit,omitempty"`
		Person      *Person      `json:"person,omitempty"`
		RoleID      int64        `json:"role_id,omitempty"`
	}

	Storage struct {
		models.Storage
	}

	Publisher struct {
		models.Publisher
		I18n map[string]*models.PublisherI18n `json:"i18n"`
	}

	SourceH struct {
		ID          int64       `json:"id"`
		UID         string      `json:"uid"`
		ParentID    null.Int64  `json:"parent_id"`
		Type        string      `json:"type"`
		Position    null.Int    `json:"position"`
		Pattern     null.String `json:"pattern,omitempty"`
		Name        null.String `json:"name"`
		Description null.String `json:"description,omitempty"`
		Children    []*SourceH  `json:"children,omitempty"`
	}

	AuthorH struct {
		Code     string      `json:"code"`
		Name     string      `json:"name"`
		FullName null.String `json:"full_name,omitempty"`
		Children []*SourceH  `json:"children,omitempty"`
	}

	TagH struct {
		ID       int64       `json:"id"`
		UID      string      `json:"uid"`
		ParentID null.Int64  `json:"parent_id"`
		Pattern  null.String `json:"pattern,omitempty"`
		Label    null.String `json:"label"`
		Children []*TagH     `json:"children,omitempty"`
	}

	ContentUnitAutonameRequest struct {
		TypeID        int64       `json:"typeId"`
		CollectionUID null.String `json:"collectionUid,omitempty"`
	}
)

func NewCollectionsResponse() *CollectionsResponse {
	return &CollectionsResponse{Collections: make([]*Collection, 0)}
}

func NewContentUnitsResponse() *ContentUnitsResponse {
	return &ContentUnitsResponse{ContentUnits: make([]*ContentUnit, 0)}
}

func NewFilesResponse() *FilesResponse {
	return &FilesResponse{Files: make([]*MFile, 0)}
}

func NewMFile(f *models.File) *MFile {
	x := &MFile{File: *f}
	if f.Sha1.Valid {
		x.Sha1Str = hex.EncodeToString(f.Sha1.Bytes)
	}
	return x
}

func NewOperationsResponse() *OperationsResponse {
	return &OperationsResponse{Operations: make([]*models.Operation, 0)}
}

func NewSourcesResponse() *SourcesResponse {
	return &SourcesResponse{Sources: make([]*Source, 0)}
}

func NewTagsResponse() *TagsResponse {
	return &TagsResponse{Tags: make([]*Tag, 0)}
}

func NewPersonsResponse() *PersonsResponse {
	return &PersonsResponse{Persons: make([]*Person, 0)}
}

func NewStoragesResponse() *StoragesResponse {
	return &StoragesResponse{Storages: make([]*models.Storage, 0)}
}

func NewPublishersResponse() *PublishersResponse {
	return &PublishersResponse{Publishers: make([]*Publisher, 0)}
}

func (mf MaybeFile) AsFile() File {
	return File{
		FileName:  mf.FileName,
		Sha1:      mf.Sha1,
		Size:      mf.Size,
		CreatedAt: mf.CreatedAt,
		Type:      mf.Type,
		SubType:   mf.SubType,
		MimeType:  mf.MimeType,
		Language:  mf.Language,
	}
}

func (drf *DateRangeFilter) Range() (time.Time, time.Time, error) {
	var err error
	var s, e time.Time

	if drf.StartDate != "" {
		s, err = time.Parse("2006-01-02", drf.StartDate)
	}
	if err == nil && drf.EndDate != "" {
		e, err = time.Parse("2006-01-02", drf.EndDate)
		if err == nil {
			e = e.Add(24*time.Hour - 1) // make the hour 23:59:59.999999999
		}
	}

	return s, e, err
}

// A time.Time like structure with Unix timestamp JSON marshalling
type Timestamp struct {
	time.Time
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	ts := t.Time.Unix()
	stamp := fmt.Sprint(ts)
	return []byte(stamp), nil
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	ts, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}
	t.Time = time.Unix(int64(ts), 0)
	return nil
}

// A time.Time like structure with date part only JSON marshalling
type Date struct {
	time.Time
}

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", d.Time.Format("2006-01-02"))), nil
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var err error
	d.Time, err = time.Parse("2006-01-02", strings.Trim(string(b), "\""))
	if err != nil {
		fmt.Println(err)
	}
	return err
}
