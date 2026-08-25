package migration

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-core/internal/models"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/database/gorm/migration"
	"gorm.io/gorm"
)

type migrationDescriptionKey struct {
	Module     string
	DataSource string
	Version    string
}

type i18nRecordKey struct {
	TargetID int64
	Locale   string
}

// syncDescriptionTranslations 将迁移 README 译文写入统一翻译表。
func syncDescriptionTranslations(ctx context.Context, databases map[string]*databaseGorm.Client, translations []DescriptionTranslation) error {
	if len(translations) == 0 {
		return nil
	}
	centralClient, ok := databases[migration.DefaultTarget]
	if !ok || centralClient == nil || centralClient.DB == nil {
		return fmt.Errorf("迁移说明翻译需要默认数据库客户端")
	}
	err := centralClient.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		histories := make([]models.BaseMigration, 0)
		result := tx.Find(&histories)
		if result.Error != nil {
			return fmt.Errorf("读取迁移记录失败: %w", result.Error)
		}
		historyIDs := make(map[migrationDescriptionKey]int64, len(histories))
		for _, history := range histories {
			historyIDs[migrationDescriptionKey{
				Module:     history.Module,
				DataSource: history.DataSource,
				Version:    history.Version,
			}] = history.ID
		}

		records := make([]models.BaseI18N, 0)
		result = tx.Where(&models.BaseI18N{TargetType: DescriptionTargetType}).Find(&records)
		if result.Error != nil {
			return fmt.Errorf("读取迁移说明译文失败: %w", result.Error)
		}
		existing := make(map[i18nRecordKey]*models.BaseI18N, len(records))
		for index := range records {
			record := &records[index]
			existing[i18nRecordKey{TargetID: record.TargetID, Locale: record.Locale}] = record
		}

		for _, translation := range translations {
			historyID := historyIDs[migrationDescriptionKey{
				Module:     translation.Module,
				DataSource: translation.DataSource,
				Version:    translation.Version,
			}]
			if historyID == 0 {
				return fmt.Errorf("迁移记录不存在: %s/%s/%s", translation.Module, translation.DataSource, translation.Version)
			}
			key := i18nRecordKey{TargetID: historyID, Locale: translation.Locale}
			record := existing[key]
			if record == nil {
				record = &models.BaseI18N{
					TargetType: DescriptionTargetType,
					TargetID:   historyID,
					Locale:     translation.Locale,
					Name:       translation.Description,
				}
				result = tx.Create(record)
				if result.Error != nil {
					return fmt.Errorf("写入迁移说明译文失败: %w", result.Error)
				}
				existing[key] = record
				continue
			}
			if record.Name == translation.Description {
				continue
			}
			record.Name = translation.Description
			result = tx.Save(record)
			if result.Error != nil {
				return fmt.Errorf("更新迁移说明译文失败: %w", result.Error)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("同步迁移说明译文: %w", err)
	}
	return nil
}
