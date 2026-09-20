package ocr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const defaultLanguages = "chi_sim+eng"

type Processor interface {
	Recognize(context.Context, []byte) (string, error)
}

type Tesseract struct {
	bin       string
	languages string
}

func NewTesseract() *Tesseract {
	bin := strings.TrimSpace(os.Getenv("OCR_TESSERACT_BIN"))
	if bin == "" {
		bin = "tesseract"
	}
	languages := strings.TrimSpace(os.Getenv("OCR_LANGUAGES"))
	if languages == "" {
		languages = defaultLanguages
	}
	return &Tesseract{bin: bin, languages: languages}
}

func (t *Tesseract) Recognize(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("OCR 输入图片为空")
	}
	file, err := os.CreateTemp("", "r1rpc-ocr-*")
	if err != nil {
		return "", fmt.Errorf("创建 OCR 临时文件失败: %w", err)
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("写入 OCR 临时文件失败: %w", err)
	}
	if err = file.Close(); err != nil {
		return "", fmt.Errorf("关闭 OCR 临时文件失败: %w", err)
	}
	output, err := exec.CommandContext(ctx, t.bin, path, "stdout", "-l", t.languages, "--psm", "6").CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("Tesseract OCR 失败: %s", message)
	}
	return strings.TrimSpace(string(output)), nil
}
