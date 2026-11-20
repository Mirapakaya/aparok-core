package api

import (
	"context"

	"github.com/anyproto/any-sync/app"

	"github.com/anyproto/anytype-heart/pkg/lib/logging"
	"github.com/anyproto/anytype-heart/pb"
)

const (
	CName           = "api"
	readTimeout     = 5 // seconds, kept for compatibility
	shutdownTimeout = 1 // millisecond, kept for compatibility
)

var log = logging.Logger("api-service")

type Service interface {
	app.ComponentRunnable
	// ReassignAddress rebinds the JSON API to listenAddr, returning the bind
	// outcome directly to the caller.
	ReassignAddress(ctx context.Context, listenAddr string) (*pb.EventAccountJsonApiStatus, error)
	RevokeToken(token string)
}

type apiService struct{}

func New() Service {
	return &apiService{}
}

func (s *apiService) Name() (name string) {
	return CName
}

func (s *apiService) Init(a *app.App) error {
	return nil
}

func (s *apiService) Run(ctx context.Context) error {
	return nil
}

func (s *apiService) Close(ctx context.Context) error {
	return nil
}

func (s *apiService) ReassignAddress(ctx context.Context, listenAddr string) (*pb.EventAccountJsonApiStatus, error) {
	return &pb.EventAccountJsonApiStatus{Success: false, ListenAddr: listenAddr, Error: "JSON API disabled in offline build"}, nil
}

func (s *apiService) RevokeToken(token string) {
}

// SetMiddlewareParams is a no-op in offline builds.
func SetMiddlewareParams(mw interface{}) {
}
