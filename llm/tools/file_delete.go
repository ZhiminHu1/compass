package tools

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// DeleteFileParams defines parameters for deleting a file.
type DeleteFileParams struct {
	Path string `json:"path" jsonschema:"description=The path of the file to delete"`
}

// deleteDescription is the detailed tool description for the AI
const deleteDescription = `Delete a specific file from the filesystem.

BEFORE USING:
- Verify the file path is correct
- Ensure you have the right to delete this file

CAPABILITIES:
- Delete individual files
- Cannot delete directories (use bash tool for that)
- Protected files cannot be deleted (.env, .git)

PARAMETERS:
- path (required): The path of the file to delete

OUTPUT FORMAT:
Returns confirmation with the file path deleted.

EXAMPLES:
- Delete file: {"path": "temp.txt"}
- Delete specific: {"path": "output.log"}

SECURITY:
- Deleting .env, .git files is blocked`

// DeleteFileFunc deletes a file.
func DeleteFileFunc(ctx context.Context, params DeleteFileParams) (string, error) {
	// Security check
	base := filepath.Base(params.Path)
	if base == ".env" || base == ".git" {
		return Error(fmt.Sprintf("deleting %s is not allowed for security reasons", base))
	}

	err := os.Remove(params.Path)
	if err != nil {
		return Error(fmt.Sprintf("failed to delete file: %v", err))
	}

	absPath, _ := filepath.Abs(params.Path)
	return Success(fmt.Sprintf("File deleted: %s", absPath), &Metadata{FilePath: absPath})
}

// GetDeleteFileTool returns the delete file tool.
func GetDeleteFileTool() tool.InvokableTool {
	t, err := utils.InferTool(DeleteToolName, deleteDescription, DeleteFileFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
