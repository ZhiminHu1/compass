package subagent

import (
	"context"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type BookSearchInput struct {
	Genre     string `json:"genre" jsonschema_description:"Preferred book genre,enum=fiction,enum=sci-fi,enum=mystery,enum=biography,enum=business"`
	MaxPages  int    `json:"max_pages" jsonschema_description:"Maximum page length (0 for no limit)"`
	MinRating int    `json:"min_rating" jsonschema_description:"Minimum user rating (0-5 scale)"`
}

type BookSearchOutput struct {
	Books []string
}

func NewBookRecommender() tool.InvokableTool {
	bookSearchTool, err := utils.InferTool(
		"search_book",
		"Search books based on user preferences,",
		func(ctx context.Context, input BookSearchInput) (output *BookSearchOutput, err error) {
			return &BookSearchOutput{Books: []string{"Pride and Prejudice"}}, nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create search book tool: %v", err)
	}
	return bookSearchTool
}
