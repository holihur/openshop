package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CategoryRepository implements port.CategoryRepository.
type CategoryRepository struct{ db *DB }

func NewCategoryRepository(db *DB) *CategoryRepository { return &CategoryRepository{db: db} }

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	m := fromCategory(c)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	c.CreatedAt, c.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	res := r.db.session(ctx).Model(&categoryModel{}).Where("id = ?", c.ID).
		Select("name", "names", "slug", "parent_id", "sort", "updated_at").
		Updates(fromCategory(c))
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	var models []categoryModel
	if err := r.db.session(ctx).Order("sort asc, created_at asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Category, 0, len(models))
	for i := range models {
		out = append(out, *toCategory(&models[i]))
	}
	return out, nil
}

func (r *CategoryRepository) FindByID(ctx context.Context, id string) (*domain.Category, error) {
	var m categoryModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toCategory(&m), nil
}

var _ port.CategoryRepository = (*CategoryRepository)(nil)
