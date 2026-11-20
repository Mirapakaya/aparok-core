package bookmarkimporter

/*
AI generated

Name: Bookmark Creation with Web Import Fallback
Scope: global

## Responsibility
- Decorates bookmark.Service to prefer web import when creating bookmark objects
- Falls back to standard CreateBookmarkObject if web import fails
*/

import (
	"context"

	"github.com/anyproto/any-sync/app"

	bookmarksvc "github.com/anyproto/anytype-heart/core/block/bookmark"
	"github.com/anyproto/anytype-heart/core/domain"
)

const CName = "bookmark-importer"

type BookmarkImporterDecorator struct {
	bookmarksvc.Service
	app.Component
}

func New() *BookmarkImporterDecorator {
	return &BookmarkImporterDecorator{}
}

func (bd *BookmarkImporterDecorator) Init(a *app.App) (err error) {
	bd.Service = a.MustComponent(bookmarksvc.CName).(bookmarksvc.Service)
	return nil
}

func (bd *BookmarkImporterDecorator) CreateBookmarkObject(ctx context.Context, spaceID, templateId string, details *domain.Details, getContent bookmarksvc.ContentFuture) (objectId string, newDetails *domain.Details, err error) {
	return bd.Service.CreateBookmarkObject(ctx, spaceID, templateId, details, getContent)
}

func (bd *BookmarkImporterDecorator) Name() (name string) {
	return CName
}
