package project

import (
	"os"
	"path/filepath"
	"strings"
)

// SharedNote is what the home of a case on a shared place says (spec
// 14.8.13).
const SharedNote = "共有の場所にある案件です（%s）。案件は手元のフォルダで持つ決まりです。手元にコピーするか、書き出し・読み込みで持ってきて使ってください（同期や、ほかの人の保存とぶつかることがあります）"

// SharedPlace tells why dir is on a shared place ("" = it is not): a
// network place (a UNC path or a network drive) or a OneDrive folder.
func SharedPlace(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	if strings.HasPrefix(abs, `\\`) || strings.HasPrefix(abs, "//") {
		return "ネットワークの場所"
	}
	if remoteDrive(abs) {
		return "ネットワークドライブ"
	}
	for _, env := range []string{"OneDrive", "OneDriveCommercial", "OneDriveConsumer"} {
		if root := os.Getenv(env); root != "" && within(abs, root) {
			return "OneDrive"
		}
	}
	return ""
}

func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
