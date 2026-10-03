package imagesearch

import (
	"context"
	"encoding/json"
	"strings"

	"r1rpc/internal/dao"
	"r1rpc/internal/model/do"
	"r1rpc/internal/model/input"
	"r1rpc/internal/model/output"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/guid"
)

func (s *Service) Create(ctx context.Context, in input.CreateImageSearchRequest) (*output.ImageSearchRequest, error) {
	if err := validateCreate(in); err != nil {
		return nil, err
	}
	if existing, err := s.findByExternalID(ctx, in.Source, in.ExternalID); err != nil || existing != nil {
		if err == nil && existing != nil && !in.DeferQueue {
			err = s.enqueuePendingRequest(ctx, existing.ID)
		}
		return existing, err
	}

	pipelineName := strings.TrimSpace(in.PipelineName)
	if pipelineName == "" {
		pipelineName = defaultPipelineName
	}
	requestID := guid.S()
	result := &output.ImageSearchRequest{
		ID: requestID, ExternalID: strings.TrimSpace(in.ExternalID), Status: statusCreated,
		PipelineName: pipelineName, PipelineVersion: defaultPipelineVersion, GroupCount: len(in.Groups),
	}
	var pendingJobs []pendingJob
	err := dao.ImageSearchRequests.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(dao.ImageSearchRequests.Table()).Ctx(ctx).Data(do.ImageSearchRequests{
			Id: requestID, Source: strings.TrimSpace(in.Source), ExternalId: nullable(in.ExternalID),
			Status: statusCreated, PipelineName: pipelineName, PipelineVersion: defaultPipelineVersion,
			Priority: in.Priority, GroupCount: len(in.Groups), RequestedByUserId: nullableInt64(in.RequestedByUserID),
			RequestedBySubject: strings.TrimSpace(in.RequestedBySubject), CallbackUrl: strings.TrimSpace(in.CallbackURL),
		}).Insert(); err != nil {
			return gerror.Wrap(err, "创建图片检索请求失败")
		}
		for groupIndex, group := range in.Groups {
			jobs, err := s.createGroup(ctx, tx, requestID, groupIndex, in.Priority, in.InitialJobStage, group)
			if err != nil {
				return err
			}
			pendingJobs = append(pendingJobs, jobs...)
		}
		return nil
	})
	if err != nil {
		if existing, lookupErr := s.findByExternalID(ctx, in.Source, in.ExternalID); lookupErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	if !in.DeferQueue {
		if err := s.enqueueJobs(ctx, requestID, pendingJobs); err != nil {
			return result, err
		}
	}
	created, err := s.GetOne(ctx, requestID)
	if err != nil {
		return result, nil
	}
	return created, nil
}

func (s *Service) createGroup(
	ctx context.Context,
	tx gdb.TX,
	requestID string,
	groupIndex int,
	priority int,
	initialJobStage string,
	in input.ImageSearchGroup,
) ([]pendingJob, error) {
	policy, err := json.Marshal(in.MatchPolicy)
	if err != nil {
		return nil, gerror.WrapCode(gcode.CodeInvalidParameter, err, "匹配策略无效")
	}
	groupID := guid.S()
	if strings.TrimSpace(initialJobStage) == "" {
		initialJobStage = statusCreated
	}
	if _, err := tx.Model(dao.ImageSearchGroups.Table()).Ctx(ctx).Data(do.ImageSearchGroups{
		Id: groupID, RequestId: requestID, ExternalId: nullable(in.ExternalID),
		SubjectUserId: strings.TrimSpace(in.SubjectUserID), Status: statusCreated,
		MatchPolicyJson: string(policy), ItemCount: len(in.Images),
	}).Insert(); err != nil {
		return nil, gerror.Wrapf(err, "创建第 %d 个图片组失败", groupIndex+1)
	}
	jobs := make([]pendingJob, 0, len(in.Images))
	for itemIndex, item := range in.Images {
		assetID, err := s.ensureAsset(ctx, tx, item.FileID)
		if err != nil {
			return nil, gerror.Wrapf(err, "图片组 %d 的第 %d 张图片无效", groupIndex+1, itemIndex+1)
		}
		var (
			itemID = guid.S()
			jobID  = guid.S()
		)
		jobInput, _ := json.Marshal(map[string]string{
			"searchRequestId": requestID, "searchGroupId": groupID, "searchItemId": itemID,
		})
		if _, err := tx.Model(dao.ImageJobs.Table()).Ctx(ctx).Data(do.ImageJobs{
			Id: jobID, SourceAssetId: assetID, Source: "image_search",
			Status: statusCreated, Stage: initialJobStage, Priority: priority, InputJson: string(jobInput),
		}).Insert(); err != nil {
			return nil, gerror.Wrap(err, "创建图片检索设备任务失败")
		}
		if _, err := tx.Model(dao.ImageSearchItems.Table()).Ctx(ctx).Data(do.ImageSearchItems{
			Id: itemID, GroupId: groupID, SourceAssetId: assetID,
			Ordinal: itemIndex, Status: statusCreated, JobId: jobID,
		}).Insert(); err != nil {
			return nil, gerror.Wrap(err, "创建图片检索项失败")
		}
		jobs = append(jobs, pendingJob{ID: jobID, ItemID: itemID, GroupID: groupID, Priority: priority})
	}
	return jobs, nil
}

func (s *Service) ensureAsset(ctx context.Context, tx gdb.TX, fileID string) (string, error) {
	var fileRecord struct {
		ID          string `orm:"id"`
		SHA256      string `orm:"sha256"`
		ContentType string `orm:"content_type"`
		SizeBytes   int64  `orm:"size_bytes"`
		ObjectKey   string `orm:"object_key"`
		Backend     string `orm:"backend"`
	}
	fileColumns := dao.Files.Columns()
	if err := tx.Model(dao.Files.Table()).Ctx(ctx).
		Fields(fileColumns.Id, fileColumns.Sha256, fileColumns.ContentType, fileColumns.SizeBytes, fileColumns.ObjectKey, fileColumns.Backend).
		Where(fileColumns.Id, strings.TrimSpace(fileID)).Where(fileColumns.Status, "active").Scan(&fileRecord); err != nil {
		return "", gerror.WrapCode(gcode.CodeNotFound, err, "图片文件不存在")
	}
	if fileRecord.ID == "" {
		return "", gerror.NewCode(gcode.CodeNotFound, "图片文件不存在")
	}
	assetColumns := dao.ImageAssets.Columns()
	assetID, err := tx.Model(dao.ImageAssets.Table()).Ctx(ctx).
		Where(assetColumns.Sha256, fileRecord.SHA256).Value(assetColumns.Id)
	if err != nil {
		return "", gerror.Wrap(err, "查询图片资产失败")
	}
	if !assetID.IsEmpty() {
		_, err = tx.Model(dao.ImageAssets.Table()).Ctx(ctx).
			Where(assetColumns.Id, assetID.String()).
			Data(do.ImageAssets{
				FileId: fileRecord.ID, MimeType: fileRecord.ContentType,
				SizeBytes: fileRecord.SizeBytes, ObjectKey: fileRecord.ObjectKey,
			}).Update()
		if err != nil {
			return "", gerror.Wrap(err, "刷新图片资产存储引用失败")
		}
		return assetID.String(), nil
	}
	newAssetID := guid.S()
	storageStatus := "available"
	if fileRecord.Backend == "file" {
		storageStatus = "local_pending"
	}
	if _, err := tx.Model(dao.ImageAssets.Table()).Ctx(ctx).Data(do.ImageAssets{
		Id: newAssetID, FileId: fileRecord.ID, Sha256: fileRecord.SHA256, MimeType: fileRecord.ContentType,
		SizeBytes: fileRecord.SizeBytes, ObjectKey: fileRecord.ObjectKey, StorageStatus: storageStatus,
	}).InsertIgnore(); err != nil {
		return "", gerror.Wrap(err, "创建图片资产失败")
	}
	assetID, err = tx.Model(dao.ImageAssets.Table()).Ctx(ctx).
		Where(assetColumns.Sha256, fileRecord.SHA256).Value(assetColumns.Id)
	if err != nil || assetID.IsEmpty() {
		return "", gerror.Wrap(err, "读取图片资产失败")
	}
	return assetID.String(), nil
}

func nullable(value string) any {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return nil
}

func nullableInt64(value int64) any {
	if value != 0 {
		return value
	}
	return nil
}
