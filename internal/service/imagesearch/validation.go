package imagesearch

import (
	"net/url"
	"strings"

	"r1rpc/internal/model/input"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

func validateCreate(in input.CreateImageSearchRequest) error {
	if strings.TrimSpace(in.Source) == "" {
		return gerror.NewCode(gcode.CodeInvalidParameter, "任务来源不能为空")
	}
	if len(strings.TrimSpace(in.ExternalID)) > 128 {
		return gerror.NewCode(gcode.CodeInvalidParameter, "externalId 长度不能超过 128")
	}
	if in.Priority < -10 || in.Priority > 10 {
		return gerror.NewCode(gcode.CodeInvalidParameter, "priority 必须在 -10 到 10 之间")
	}
	if len(strings.TrimSpace(in.PipelineName)) > 64 {
		return gerror.NewCode(gcode.CodeInvalidParameter, "pipelineName 长度不能超过 64")
	}
	if err := validateCallbackURL(in.CallbackURL); err != nil {
		return err
	}
	if len(in.Groups) == 0 || len(in.Groups) > 1000 {
		return gerror.NewCode(gcode.CodeInvalidParameter, "图片组数量必须在 1 到 1000 之间")
	}
	for groupIndex, group := range in.Groups {
		if len(strings.TrimSpace(group.ExternalID)) > 128 || len(strings.TrimSpace(group.SubjectUserID)) > 128 {
			return gerror.NewCodef(gcode.CodeInvalidParameter, "第 %d 个图片组的标识长度不能超过 128", groupIndex+1)
		}
		if len(group.Images) == 0 || len(group.Images) > 100 {
			return gerror.NewCodef(gcode.CodeInvalidParameter, "第 %d 个图片组的图片数量必须在 1 到 100 之间", groupIndex+1)
		}
		seen := make(map[string]struct{}, len(group.Images))
		for _, image := range group.Images {
			fileID := strings.TrimSpace(image.FileID)
			if len(fileID) != 32 {
				return gerror.NewCodef(gcode.CodeInvalidParameter, "第 %d 个图片组包含无效 fileId", groupIndex+1)
			}
			if _, ok := seen[fileID]; ok {
				return gerror.NewCodef(gcode.CodeInvalidParameter, "第 %d 个图片组包含重复图片", groupIndex+1)
			}
			seen[fileID] = struct{}{}
		}
	}
	return nil
}

func validateCallbackURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if len(raw) > 1024 {
		return gerror.NewCode(gcode.CodeInvalidParameter, "callbackUrl 长度不能超过 1024")
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return gerror.NewCode(gcode.CodeInvalidParameter, "callbackUrl 必须是有效的 HTTP(S) 地址")
	}
	return nil
}
