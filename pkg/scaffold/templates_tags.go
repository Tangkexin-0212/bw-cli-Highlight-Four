package scaffold

const serviceGormRepoTemplate = `package repo

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"{{ .Module }}/internal/{{ .Dir }}/model"
)

// {{ .Pascal }}Model is the Gorm persistence model for the {{ .TableName }} table.
type {{ .Pascal }}Model struct {
	ID          string ` + "`gorm:\"primaryKey;size:64\"`" + `
	Name        string ` + "`gorm:\"size:128;not null\"`" + `
	Description string ` + "`gorm:\"type:text\"`" + `
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func ({{ .Pascal }}Model) TableName() string {
	return "{{ .TableName }}"
}

// GormRepository persists {{ .Dir }} aggregates with Gorm.
type GormRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewGormRepository constructs a {{ .Dir }} repository with optional structured logging.
func NewGormRepository(db *gorm.DB, loggers ...*zap.Logger) *GormRepository {
	log := zap.NewNop()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return &GormRepository{db: db, log: log}
}

// AutoMigrate creates or updates the {{ .TableName }} table schema.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&{{ .Pascal }}Model{})
}

// Save inserts or updates a {{ .Dir }} aggregate.
func (r *GormRepository) Save(ctx context.Context, item *model.{{ .Pascal }}) error {
	start := time.Now()
	tx := r.db.WithContext(ctx).Save(toRecord(item))
	r.logOperation("Save", tx.RowsAffected, start, tx.Error)
	return tx.Error
}

// FindByID loads a {{ .Dir }} aggregate by id.
func (r *GormRepository) FindByID(ctx context.Context, id string) (*model.{{ .Pascal }}, error) {
	start := time.Now()
	var record {{ .Pascal }}Model
	tx := r.db.WithContext(ctx).Where("id = ?", id).First(&record)
	err := tx.Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = model.Err{{ .Pascal }}NotFound
	}
	if err != nil {
		r.logOperation("FindByID", tx.RowsAffected, start, err)
		return nil, err
	}
	r.logOperation("FindByID", tx.RowsAffected, start, nil)
	return toDomain(&record), nil
}

// List loads paginated {{ .Dir }} aggregates.
func (r *GormRepository) List(ctx context.Context, offset int, limit int) ([]*model.{{ .Pascal }}, int64, error) {
	start := time.Now()
	var total int64
	countTx := r.db.WithContext(ctx).Model(&{{ .Pascal }}Model{}).Count(&total)
	if countTx.Error != nil {
		r.logOperation("Count", countTx.RowsAffected, start, countTx.Error)
		return nil, 0, countTx.Error
	}
	var records []{{ .Pascal }}Model
	tx := r.db.WithContext(ctx).
		Order("created_at desc").
		Offset(offset).
		Limit(limit).
		Find(&records)
	if tx.Error != nil {
		r.logOperation("List", tx.RowsAffected, start, tx.Error)
		return nil, 0, tx.Error
	}
	items := make([]*model.{{ .Pascal }}, 0, len(records))
	for i := range records {
		items = append(items, toDomain(&records[i]))
	}
	r.logOperation("List", tx.RowsAffected, start, nil)
	return items, total, nil
}

// Delete removes a {{ .Dir }} aggregate by id.
func (r *GormRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	tx := r.db.WithContext(ctx).Where("id = ?", id).Delete(&{{ .Pascal }}Model{})
	err := tx.Error
	if err == nil && tx.RowsAffected == 0 {
		err = model.Err{{ .Pascal }}NotFound
	}
	r.logOperation("Delete", tx.RowsAffected, start, err)
	return err
}

func (r *GormRepository) logOperation(operation string, rows int64, start time.Time, err error) {
	fields := []zap.Field{
		zap.String("repository", "{{ .Dir }}"),
		zap.String("operation", operation),
		zap.Int64("rows_affected", rows),
		zap.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		r.log.Warn("repository operation completed with error", fields...)
		return
	}
	r.log.Info("repository operation completed", fields...)
}

func toRecord(item *model.{{ .Pascal }}) *{{ .Pascal }}Model {
	return &{{ .Pascal }}Model{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func toDomain(record *{{ .Pascal }}Model) *model.{{ .Pascal }} {
	return &model.{{ .Pascal }}{
		ID:          record.ID,
		Name:        record.Name,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

var _ model.Repository = (*GormRepository)(nil)
`

const gatewayRequestTemplate = `package request

// Create{{ .Pascal }}Request is the JSON payload used by POST /api/v1/{{ .TableName }}.
type Create{{ .Pascal }}Request struct {
	Name        string ` + "`json:\"name\" binding:\"required\"`" + `
	Description string ` + "`json:\"description\"`" + `
}

// Update{{ .Pascal }}Request is the JSON payload used by PUT /api/v1/{{ .TableName }}/:id.
type Update{{ .Pascal }}Request struct {
	Name        string ` + "`json:\"name\" binding:\"required\"`" + `
	Description string ` + "`json:\"description\"`" + `
}

// List{{ .Pascal }}Request is the query string payload used by GET /api/v1/{{ .TableName }}.
type List{{ .Pascal }}Request struct {
	Page     int32 ` + "`form:\"page\"`" + `
	PageSize int32 ` + "`form:\"page_size\"`" + `
}
`
