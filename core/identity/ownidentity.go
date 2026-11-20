package identity

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/anyproto/any-sync/util/crypto"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-heart/core/anytype/account"
	"github.com/anyproto/anytype-heart/core/domain"
	"github.com/anyproto/anytype-heart/core/files/fileacl"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/database"
	"github.com/anyproto/anytype-heart/pkg/lib/localstore/objectstore"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/space"
	"github.com/anyproto/anytype-heart/space/clientspace"
)

type observerService interface {
	broadcastMyIdentityProfile(identityProfile *model.IdentityProfile)
	refreshMyIdentityProfile()
}

var errOwnProfileNotLoaded = errors.New("own profile details are not loaded")

type ownProfileSubscription struct {
	spaceService    space.Service
	objectStore     objectstore.ObjectStore
	accountService  account.Service
	fileAclService  fileacl.Service
	observerService observerService

	myIdentity   string
	gotDetailsCh chan struct{}

	detailsLock sync.Mutex
	gotDetails  bool
	details     *domain.Details

	componentCtx       context.Context
	componentCtxCancel context.CancelFunc
}

func newOwnProfileSubscription(
	spaceService space.Service,
	objectStore objectstore.ObjectStore,
	accountService account.Service,
	fileAclService fileacl.Service,
	observerService observerService,
) *ownProfileSubscription {
	componentCtx, componentCtxCancel := context.WithCancel(context.Background())
	return &ownProfileSubscription{
		spaceService:       spaceService,
		objectStore:        objectStore,
		accountService:       accountService,
		fileAclService:     fileAclService,
		observerService:    observerService,
		gotDetailsCh:       make(chan struct{}),
		componentCtx:       componentCtx,
		componentCtxCancel: componentCtxCancel,
	}
}

func (s *ownProfileSubscription) run(ctx context.Context) (err error) {
	s.myIdentity = s.accountService.AccountID()
	techSpace, err := s.spaceService.GetTechSpace(ctx)
	if err != nil {
		return fmt.Errorf("get space: %w", err)
	}
	id, err := techSpace.(*clientspace.TechSpace).TechSpace.AccountObjectId()
	if err != nil {
		return err
	}

	recordsCh := make(chan *domain.Details)
	sub := database.NewSubscription(nil, recordsCh)

	var (
		records  []database.Record
		closeSub func()
	)

	records, closeSub, err = s.objectStore.SpaceIndex(s.spaceService.TechSpaceId()).QueryByIdsAndSubscribeForChanges([]string{id}, sub)
	if err != nil {
		return err
	}
	go func() {
		select {
		case <-s.componentCtx.Done():
			closeSub()
			return
		}
	}()

	if len(records) > 0 {
		s.handleOwnProfileDetails(records[0].Details)
	}

	go func() {
		for {
			select {
			case <-s.componentCtx.Done():
				return
			case rec, ok := <-recordsCh:
				if !ok {
					return
				}
				s.handleOwnProfileDetails(rec)
			}
		}
	}()

	return nil
}

func (s *ownProfileSubscription) close() {
	s.componentCtxCancel()
}

// enqueuePush is a no-op in offline builds.
func (s *ownProfileSubscription) enqueuePush() {
}

func (s *ownProfileSubscription) handleOwnProfileDetails(profileDetails *domain.Details) {
	if profileDetails == nil {
		return
	}
	s.detailsLock.Lock()
	if !s.gotDetails {
		close(s.gotDetailsCh)
		s.gotDetails = true
	}

	if s.details == nil {
		s.details = domain.NewDetails()
	}
	for _, key := range []domain.RelationKey{
		bundle.RelationKeyId,
		bundle.RelationKeyName,
		bundle.RelationKeyDescription,
		bundle.RelationKeyGlobalName,
	} {
		if v, ok := profileDetails.TryString(key); ok {
			s.details.SetString(key, v)
		}
	}
	if iconObjectId, ok := profileDetails.TryString(bundle.RelationKeyIconImage); ok {
		iconCid, _, err := s.prepareIconImageInfo(iconObjectId)
		if err != nil {
			log.Error("handleOwnProfileDetails: resolve icon to file cid", zap.Error(err))
			iconCid = iconObjectId
		}
		s.details.SetString(bundle.RelationKeyIconImage, iconCid)
	}
	identityProfile := s.prepareIdentityProfile()
	s.detailsLock.Unlock()

	s.observerService.broadcastMyIdentityProfile(identityProfile)
}

func (s *ownProfileSubscription) updateGlobalName(globalName string) {
}

func (s *ownProfileSubscription) prepareIdentityProfile() *model.IdentityProfile {
	return &model.IdentityProfile{
		Identity:    s.myIdentity,
		Name:        s.details.GetString(bundle.RelationKeyName),
		Description: s.details.GetString(bundle.RelationKeyDescription),
		IconCid:     s.details.GetString(bundle.RelationKeyIconImage),
	}
}

// isLoaded reports whether the own profile details have been loaded
func (s *ownProfileSubscription) isLoaded() bool {
	s.detailsLock.Lock()
	defer s.detailsLock.Unlock()
	return s.gotDetails
}

// prepareOwnIdentityProfile returns the own profile as it is kept locally.
func (s *ownProfileSubscription) prepareOwnIdentityProfile() (*model.IdentityProfile, error) {
	s.detailsLock.Lock()
	defer s.detailsLock.Unlock()
	if !s.gotDetails {
		return nil, errOwnProfileNotLoaded
	}

	iconImageObjectId := s.details.GetString(bundle.RelationKeyIconImage)
	iconCid, iconEncryptionKeys, err := s.prepareIconImageInfo(iconImageObjectId)
	if err != nil {
		return nil, fmt.Errorf("prepare icon image info: %w", err)
	}

	identity := s.accountService.AccountID()
	return &model.IdentityProfile{
		Identity:           identity,
		Name:               s.details.GetString(bundle.RelationKeyName),
		Description:        s.details.GetString(bundle.RelationKeyDescription),
		IconCid:            iconCid,
		IconEncryptionKeys: iconEncryptionKeys,
	}, nil
}

func (s *ownProfileSubscription) prepareIconImageInfo(iconImageObjectId string) (iconCid string, iconEncryptionKeys []*model.FileEncryptionKey, err error) {
	if iconImageObjectId == "" {
		return "", nil, nil
	}
	return s.fileAclService.GetInfoForFileSharing(iconImageObjectId)
}

func (s *ownProfileSubscription) getDetails(ctx context.Context) (identity string, metadataKey crypto.SymKey, details *domain.Details) {
	select {
	case <-s.gotDetailsCh:

	case <-ctx.Done():
		return "", nil, nil
	case <-s.componentCtx.Done():
		return "", nil, nil
	}
	s.detailsLock.Lock()
	defer s.detailsLock.Unlock()

	detailsCopy := s.details.Copy()
	return s.myIdentity, s.spaceService.AccountMetadataSymKey(), detailsCopy
}
