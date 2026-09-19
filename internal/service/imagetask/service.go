package imagetask

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"r1rpc/internal/dao"
	"r1rpc/internal/model"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/input"
	"r1rpc/internal/model/output"
	"r1rpc/internal/store"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/guid"
)

const (
	maxBatchFiles = 100
	statusCreated = "created"
)

// Service 提供与接入协议无关的单图和批量任务创建能力。
type Service struct {
	legacyStore *store.Store
	queueMu     sync.RWMutex
	enqueuer    Enqueuer
}

func New(legacyStore *store.Store) *Service {
	return &Service{legacyStore: legacyStore}
}

// CreateJob 创建一个独立单图任务。
func (s *Service) CreateJob(ctx context.Context, in input.CreateImageJob) (*output.ImageJob, error) {
	if err := validateSourceAndFiles(in.Source, []string{in.FileID}); err != nil {
		return nil, err
	}
	if existing, err := findExistingJob(ctx, in.Source, in.ExternalID); err != nil || existing != nil {
		return existing, err
	}
	var result *output.ImageJob
	err := dao.ImageJobs.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		job, err := s.createJobInTransaction(ctx, tx, "", in.Source, in.ExternalID, in.FileID,
			in.Priority, in.ForceRefresh)
		if err != nil {
			return err
		}
		result = job
		return nil
	})
	if err == nil && result != nil {
		err = s.enqueue(ctx, result, in.Priority)
	}
	return result, err
}

// CreateBatch 在一个事务中创建批次及所有子任务。
func (s *Service) CreateBatch(ctx context.Context, in input.CreateImageBatch) (*output.ImageBatch, error) {
	if err := validateSourceAndFiles(in.Source, in.FileIDs); err != nil {
		return nil, err
	}
	if existing, err := findExistingBatch(ctx, in.Source, in.ExternalID); err != nil || existing != nil {
		return existing, err
	}

	requestedBy, err := json.Marshal(in.RequestedBy)
	if err != nil {
		return nil, gerror.WrapCode(gcode.CodeInvalidParameter, err, "序列化任务创建者失败")
	}
	batchID := guid.S()
	result := &output.ImageBatch{ID: batchID, Status: statusCreated, Total: len(in.FileIDs)}
	err = dao.ImageBatches.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(dao.ImageBatches.Table()).Ctx(ctx).Data(do.ImageBatches{
			Id:              batchID,
			Source:          strings.TrimSpace(in.Source),
			ExternalId:      nullableString(in.ExternalID),
			Status:          statusCreated,
			Priority:        in.Priority,
			ForceRefresh:    in.ForceRefresh,
			TotalCount:      len(in.FileIDs),
			QueuedCount:     0,
			RequestedByJson: string(requestedBy),
		}).Insert(); err != nil {
			return gerror.Wrap(err, "创建图片批次失败")
		}
		for _, fileID := range in.FileIDs {
			job, err := s.createJobInTransaction(ctx, tx, batchID, in.Source, "", fileID,
				in.Priority, in.ForceRefresh)
			if err != nil {
				return err
			}
			result.Jobs = append(result.Jobs, *job)
		}
		return nil
	})
	if err == nil {
		for index := range result.Jobs {
			if enqueueErr := s.enqueue(ctx, &result.Jobs[index], in.Priority); enqueueErr != nil {
				return result, enqueueErr
			}
		}
		if len(result.Jobs) > 0 {
			_, err = dao.ImageBatches.Ctx(ctx).Where(dao.ImageBatches.Columns().Id, batchID).
				Data(do.ImageBatches{Status: "queued", QueuedCount: len(result.Jobs)}).Update()
			result.Status = "queued"
		}
	}
	return result, err
}

func (s *Service) createJobInTransaction(
	ctx context.Context,
	tx gdb.TX,
	batchID, source, externalID, fileID string,
	priority int,
	forceRefresh bool,
) (*output.ImageJob, error) {
	assetID, err := s.ensureAsset(ctx, tx, fileID)
	if err != nil {
		return nil, err
	}
	jobID := guid.S()
	if _, err := tx.Model(dao.ImageJobs.Table()).Ctx(ctx).Data(do.ImageJobs{
		Id:            jobID,
		BatchId:       batchID,
		SourceAssetId: assetID,
		Source:        strings.TrimSpace(source),
		ExternalId:    nullableString(externalID),
		Status:        statusCreated,
		Stage:         statusCreated,
		Priority:      priority,
		ForceRefresh:  forceRefresh,
	}).Insert(); err != nil {
		return nil, gerror.Wrap(err, "创建图片任务失败")
	}
	return &output.ImageJob{ID: jobID, BatchID: batchID, Status: statusCreated}, nil
}

func (s *Service) ensureAsset(ctx context.Context, tx gdb.TX, fileID string) (string, error) {
	item, err := s.legacyStore.GetFile(ctx, strings.TrimSpace(fileID))
	if err != nil {
		return "", gerror.WrapCode(gcode.CodeNotFound, err, "图片文件不存在或不可用")
	}
	columns := dao.ImageAssets.Columns()
	record, err := tx.Model(dao.ImageAssets.Table()).Ctx(ctx).
		Where(columns.Sha256, item.SHA256).One()
	if err != nil {
		return "", gerror.Wrap(err, "查询图片资产失败")
	}
	if !record.IsEmpty() {
		return record[columns.Id].String(), nil
	}
	assetID := guid.S()
	if _, err := tx.Model(dao.ImageAssets.Table()).Ctx(ctx).Data(do.ImageAssets{
		Id:            assetID,
		FileId:        item.ID,
		Sha256:        item.SHA256,
		MimeType:      item.ContentType,
		SizeBytes:     item.SizeBytes,
		ObjectKey:     item.ObjectKey,
		StorageStatus: initialStorageStatus(*item),
	}).InsertIgnore(); err != nil {
		return "", gerror.Wrap(err, "创建图片资产失败")
	}
	// 并发任务可能同时创建相同 SHA 资产，忽略冲突后读取数据库中的权威 ID。
	record, err = tx.Model(dao.ImageAssets.Table()).Ctx(ctx).
		Where(columns.Sha256, item.SHA256).One()
	if err != nil {
		return "", gerror.Wrap(err, "读取图片资产失败")
	}
	if record.IsEmpty() {
		return "", gerror.New("图片资产创建后不存在")
	}
	return record[columns.Id].String(), nil
}

func initialStorageStatus(item model.File) string {
	if item.Backend == "file" {
		return "local_pending"
	}
	return "available"
}
